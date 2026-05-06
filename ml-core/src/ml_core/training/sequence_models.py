import torch
import torch.nn as nn

class GRUSequenceModel(nn.Module):
    def __init__(
        self,
        input_size: int,
        hidden_size: int = 64,
        num_layers: int = 2,
        num_classes: int = 3,
        dropout: float = 0.2
    ):
        super().__init__()
        self.gru = nn.GRU(
            input_size=input_size,
            hidden_size=hidden_size,
            num_layers=num_layers,
            batch_first=True,
            dropout=dropout if num_layers > 1 else 0
        )
        self.dropout = nn.Dropout(dropout)
        self.fc = nn.Linear(hidden_size, num_classes)

    def forward(self, x):
        # x shape: (batch, seq_len, input_size)
        out, _ = self.gru(x)
        # We take the output of the last time step
        out = out[:, -1, :]
        out = self.dropout(out)
        logits = self.fc(out)
        return logits

class TemporalCNNModel(nn.Module):
    def __init__(
        self,
        input_size: int,
        num_classes: int = 3,
        num_channels: list[int] = [64, 64],
        kernel_size: int = 3,
        dropout: float = 0.2
    ):
        super().__init__()
        layers = []
        in_channels = input_size
        for out_channels in num_channels:
            layers.append(nn.Conv1d(in_channels, out_channels, kernel_size, padding=(kernel_size - 1) // 2))
            layers.append(nn.BatchNorm1d(out_channels))
            layers.append(nn.ReLU())
            layers.append(nn.Dropout(dropout))
            in_channels = out_channels

        self.network = nn.Sequential(*layers)
        self.fc = nn.Linear(num_channels[-1], num_classes)

    def forward(self, x):
        # x shape: (batch, seq_len, input_size) -> (batch, input_size, seq_len)
        x = x.transpose(1, 2)
        out = self.network(x)
        # Global average pooling
        out = torch.mean(out, dim=2)
        logits = self.fc(out)
        return logits
