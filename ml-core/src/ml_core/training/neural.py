import json
import logging
from dataclasses import dataclass, field, asdict
from pathlib import Path
from typing import Any

import numpy as np
import pandas as pd
import torch
import torch.nn as nn
import torch.optim as optim
from torch.utils.data import DataLoader, Dataset
from sklearn.preprocessing import StandardScaler
from sklearn.metrics import f1_score

from ml_core.training.sequence_models import GRUSequenceModel, TemporalCNNModel
from ml_core.modeling.manifest import ModelManifest
from ml_core.modeling.export import export_to_onnx, export_to_torch_export

logger = logging.getLogger(__name__)

class TorchSequenceDataset(Dataset):
    def __init__(self, data_frames: list[pd.DataFrame], window_bars: int, feature_count: int):
        self.features = []
        self.labels = []

        for df in data_frames:
            if df.empty:
                continue
            # features is list of flattened arrays
            f = np.stack(df["features"].values).astype(np.float32)
            l = df["label"].values.astype(np.int64)
            self.features.append(f)
            self.labels.append(l)

        if self.features:
            self.features = np.concatenate(self.features, axis=0)
            self.labels = np.concatenate(self.labels, axis=0)
        else:
            self.features = np.zeros((0, window_bars * feature_count), dtype=np.float32)
            self.labels = np.zeros((0,), dtype=np.int64)

        self.window_bars = window_bars
        self.feature_count = feature_count

    def __len__(self):
        return len(self.labels)

    def __getitem__(self, idx):
        x = self.features[idx].reshape(self.window_bars, self.feature_count)
        y = self.labels[idx]
        return torch.from_numpy(x), torch.tensor(y)

@dataclass
class NeuralTrainingConfig:
    sequence_root: Path
    dataset_version: str
    window_bars: int
    output_root: Path
    model_family: str = "gru" # or "cnn"
    epochs: int = 50
    batch_size: int = 64
    learning_rate: float = 1e-3
    hidden_size: int = 64
    num_layers: int = 2
    dropout: float = 0.2
    early_stopping_patience: int = 5
    device: str = "cpu"
    random_seed: int = 42

def train_neural_sequence(config: NeuralTrainingConfig):
    torch.manual_seed(config.random_seed)
    np.random.seed(config.random_seed)

    dataset_dir = config.sequence_root / f"dataset_version={config.dataset_version}" / f"window={config.window_bars}"
    manifest_path = dataset_dir / "manifest.json"
    if not manifest_path.exists():
        raise FileNotFoundError(f"Sequence manifest not found: {manifest_path}")

    with open(manifest_path, "r") as f:
        manifest = json.load(f)

    feature_count = manifest["feature_count"]
    class_mapping = manifest["class_mapping"]

    loaders = {}
    scalers = {}

    # Load data for all splits
    all_features = []

    for split in ["train", "val", "test"]:
        split_dir = dataset_dir / f"split={split}"
        if not split_dir.exists():
            logger.warning(f"Split {split} not found in {dataset_dir}")
            continue

        dfs = []
        for p in split_dir.glob("*.parquet"):
            dfs.append(pd.read_parquet(p))

        if not dfs:
            continue

        if split == "train":
            # Collect all train features to fit scaler
            f_train = np.stack(pd.concat(dfs)["features"].values)
            scaler = StandardScaler()
            scaler.fit(f_train)
            scalers["standard"] = scaler

        # We need to apply scaling.
        # But wait, standard scaler fits on flattened.
        # For sequences, we usually scale per feature across all time steps and windows.
        # If we fit on (N, W*F), it's the same as scaling each (window_pos, feature) separately, which is NOT what we want.
        # We want to scale feature F across all windows and all time steps in windows.

        # Correct way to scale sequence features:
        # Reshape to (N*W, F), fit, transform, reshape back.

        # Let's refine the scaler fit logic.

    # Re-loading and scaling
    data_split_frames = {}
    train_df_all = []
    for split in ["train", "val", "test"]:
        split_dir = dataset_dir / f"split={split}"
        if not split_dir.exists(): continue
        dfs = [pd.read_parquet(p) for p in split_dir.glob("*.parquet")]
        if not dfs: continue
        combined = pd.concat(dfs)
        data_split_frames[split] = combined
        if split == "train":
            train_df_all.append(combined)

    if not train_df_all:
        raise ValueError("No train data found")

    train_combined = pd.concat(train_df_all)
    train_features_flat = np.stack(train_combined["features"].values) # (N, W*F)
    train_features_seq = train_features_flat.reshape(-1, feature_count) # (N*W, F)

    scaler = StandardScaler()
    scaler.fit(train_features_seq)

    for split, df in data_split_frames.items():
        f_flat = np.stack(df["features"].values)
        f_seq = f_flat.reshape(-1, feature_count)
        f_scaled = scaler.transform(f_seq)
        df["features"] = [row for row in f_scaled.reshape(-1, config.window_bars * feature_count)]

        dataset = TorchSequenceDataset([df], config.window_bars, feature_count)
        loaders[split] = DataLoader(dataset, batch_size=config.batch_size, shuffle=(split == "train"))

    # Model initialization
    if config.model_family == "gru":
        model = GRUSequenceModel(
            input_size=feature_count,
            hidden_size=config.hidden_size,
            num_layers=config.num_layers,
            num_classes=len(class_mapping),
            dropout=config.dropout
        )
    elif config.model_family == "cnn":
        model = TemporalCNNModel(
            input_size=feature_count,
            num_classes=len(class_mapping),
            dropout=config.dropout
        )
    else:
        raise ValueError(f"Unknown model family: {config.model_family}")

    model.to(config.device)

    # Class weights for imbalance
    train_labels = data_split_frames["train"]["label"].values
    class_counts = np.bincount(train_labels, minlength=len(class_mapping))
    weights = 1.0 / (class_counts + 1)
    weights = weights / weights.sum() * len(class_mapping)
    criterion = nn.CrossEntropyLoss(weight=torch.tensor(weights, dtype=torch.float32).to(config.device))

    optimizer = optim.Adam(model.parameters(), lr=config.learning_rate)

    best_val_f1 = -1.0
    patience_counter = 0
    config.output_root.mkdir(parents=True, exist_ok=True)

    for epoch in range(config.epochs):
        model.train()
        train_loss = 0.0
        for x, y in loaders["train"]:
            x, y = x.to(config.device), y.to(config.device)
            optimizer.zero_grad()
            logits = model(x)
            loss = criterion(logits, y)
            loss.backward()
            optimizer.step()
            train_loss += loss.item()

        # Validation
        model.eval()
        val_preds = []
        val_labels = []
        val_loss = 0.0
        with torch.no_grad():
            for x, y in loaders.get("val", []):
                x, y = x.to(config.device), y.to(config.device)
                logits = model(x)
                loss = criterion(logits, y)
                val_loss += loss.item()
                preds = torch.argmax(logits, dim=1)
                val_preds.extend(preds.cpu().numpy())
                val_labels.extend(y.cpu().numpy())

        if not val_preds:
            logger.warning("No validation data, skipping early stopping")
            torch.save(model.state_dict(), config.output_root / "model.pth")
            continue

        val_f1 = f1_score(val_labels, val_preds, average="macro")
        logger.info(f"Epoch {epoch}: Train Loss {train_loss/len(loaders['train']):.4f}, Val Loss {val_loss/len(loaders['val']):.4f}, Val F1 {val_f1:.4f}")

        if val_f1 > best_val_f1:
            best_val_f1 = val_f1
            torch.save(model.state_dict(), config.output_root / "model.pth")
            patience_counter = 0
        else:
            patience_counter += 1
            if patience_counter >= config.early_stopping_patience:
                logger.info("Early stopping triggered")
                break

    # Load best model for final evaluation
    if (config.output_root / "model.pth").exists():
        model.load_state_dict(torch.load(config.output_root / "model.pth"))

    # Save artifacts
    # Save scaler
    import joblib
    joblib.dump(scaler, config.output_root / "scaler.joblib")

    # Export to ONNX
    try:
        export_to_onnx(model, [config.window_bars, feature_count], config.output_root / "model.onnx")
        export_format = "onnx"
        model_artifact_path = "model.onnx"
    except Exception as e:
        logger.warning(f"ONNX export failed: {e}, falling back to torch.export")
        export_to_torch_export(model, [config.window_bars, feature_count], config.output_root / "model.pt2")
        export_format = "torch_export"
        model_artifact_path = "model.pt2"

    # Final evaluation and saving predictions
    test_metrics = {}
    for split in ["val", "test"]:
        if split not in loaders:
            continue

        model.eval()
        all_preds = []
        all_labels = []
        all_probs = []

        with torch.no_grad():
            for x, y in loaders[split]:
                x, y = x.to(config.device), y.to(config.device)
                logits = model(x)
                probs = torch.softmax(logits, dim=1)
                preds = torch.argmax(logits, dim=1)

                all_preds.extend(preds.cpu().numpy())
                all_labels.extend(y.cpu().numpy())
                all_probs.append(probs.cpu().numpy())

        if not all_preds:
            continue

        probs_concat = np.concatenate(all_probs, axis=0)

        pred_df = pd.DataFrame({
            "label": all_labels,
            "prediction": all_preds,
            "prob_down": probs_concat[:, 0],
            "prob_no_trade": probs_concat[:, 1],
            "prob_up": probs_concat[:, 2]
        })
        pred_df.to_parquet(config.output_root / f"predictions_{split}.parquet", index=False)

        if split == "test":
            test_f1 = f1_score(all_labels, all_preds, average="macro")
            test_metrics["f1_macro"] = float(test_f1)

    # Create manifest
    import hashlib
    def get_sha256(file_path):
        if not file_path.exists(): return ""
        with open(file_path, "rb") as f:
            return hashlib.sha256(f.read()).hexdigest()

    model_sha = get_sha256(config.output_root / model_artifact_path)

    manifest = ModelManifest(
        model_version=config.output_root.name,
        model_family=config.model_family,
        task="multiclass",
        classes=["down", "no_trade", "up"],
        input_timeframe=manifest["timeframe"],
        prediction_horizon_bars=manifest["horizon_bars"],
        feature_schema_version="sprint8",
        feature_order=manifest["feature_order"],
        export_format=export_format,
        model_artifact_path=model_artifact_path,
        artifact_sha256=model_sha,
        metrics=test_metrics,
        threshold=0.33,
        calibration="none",
        created_at=pd.Timestamp.now(tz="UTC").isoformat().replace("+00:00", "Z"),
        source_dataset_version=config.dataset_version,
        input_window_bars=config.window_bars,
        input_tensor_shape=[config.window_bars, feature_count],
        normalization="standard"
    )
    manifest.save(config.output_root / "model_manifest.json")

    return manifest
