import torch
import torch.nn as nn
from pathlib import Path
import logging

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

def export_to_torchscript(model: nn.Module, input_shape: list[int], output_path: Path):
    model.eval()
    dummy_input = torch.randn(1, *input_shape)
    traced_model = torch.jit.trace(model, dummy_input)
    traced_model.save(str(output_path))
    logger.info(f"Model exported to {output_path}")
