import torch
import torch.nn as nn
from pathlib import Path
import logging
import warnings

logger = logging.getLogger(__name__)

def export_to_onnx(model: nn.Module, input_shape: list[int], output_path: Path):
    model.eval()
    dummy_input = torch.randn(1, *input_shape)

    torch.onnx.export(
        model,
        dummy_input,
        str(output_path),
        export_params=True,
        opset_version=14,
        do_constant_folding=True,
        input_names=["input"],
        output_names=["output"],
        dynamic_axes={"input": {0: "batch_size"}, "output": {0: "batch_size"}}
    )
    logger.info(f"Model exported to {output_path}")

def export_to_torch_export(model: nn.Module, input_shape: list[int], output_path: Path):
    model.eval()
    dummy_input = torch.randn(1, *input_shape)
    with warnings.catch_warnings():
        warnings.filterwarnings("ignore", message="The tensor attributes self\\.gru\\._flat_weights.*", category=UserWarning)
        exported_model = torch.export.export(model, (dummy_input,))
    torch.export.save(exported_model, str(output_path))
    logger.info(f"Model exported to {output_path}")
