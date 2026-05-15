import React, { useState } from "react";
import { Bell, X } from "lucide-react";

interface AlertModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (rule: any) => void;
  ticker: string;
}

export const AlertModal: React.FC<AlertModalProps> = ({ isOpen, onClose, onSubmit, ticker }) => {
  const [eventType, setEventType] = useState("decision_threshold_triggered");
  const [threshold, setThreshold] = useState("0.65");
  const [severity, setSeverity] = useState("warning");
  const [direction, setDirection] = useState("");
  const [cooldownMinutes, setCooldownMinutes] = useState("60");

  if (!isOpen) return null;

  const handleSubmit = () => {
    const probability = Number(threshold.replace(",", "."));
    const normalizedThreshold = probability > 1 && probability <= 100 ? probability / 100 : probability;

    onSubmit({
      ticker,
      event_type: eventType,
      threshold: eventType === "decision_threshold_triggered" ? normalizedThreshold : undefined,
      severity,
      direction: direction || undefined,
      trigger_mode: "once",
      delivery_channels: ["app"],
      is_enabled: true,
      cooldown_minutes: Number.parseInt(cooldownMinutes, 10) || 60,
    });
  };

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal-content advanced-alert-modal" onClick={(event) => event.stopPropagation()}>
        <div className="modal-header">
          <h2><Bell size={20} /> Уведомление для <strong>{ticker}</strong></h2>
          <button type="button" className="modal-close" onClick={onClose} title="Закрыть"><X size={20} /></button>
        </div>

        <div className="modal-body">
          <div className="alert-form-row">
            <label>Событие</label>
            <select
              value={eventType}
              onChange={(event) => {
                const nextType = event.target.value;
                setEventType(nextType);
                if (nextType !== "decision_threshold_triggered") {
                  setThreshold("");
                } else if (!threshold) {
                  setThreshold("0.65");
                }
              }}
            >
              <option value="decision_threshold_triggered">Сильный ML-прогноз</option>
              <option value="classification_success">Любой ML-прогноз</option>
              <option value="inference_blocked_by_runtime">Ошибка расчета</option>
            </select>
          </div>

          <div className="alert-form-row">
            <label>Направление</label>
            <select value={direction} onChange={(event) => setDirection(event.target.value)}>
              <option value="">Любое</option>
              <option value="up">Рост</option>
              <option value="down">Снижение</option>
            </select>
          </div>

          <div className="alert-form-row">
            <label>Порог вероятности</label>
            <input
              inputMode="decimal"
              placeholder={eventType === "decision_threshold_triggered" ? "0.65 или 65" : "не требуется"}
              value={threshold}
              disabled={eventType !== "decision_threshold_triggered"}
              onChange={(event) => setThreshold(event.target.value)}
            />
          </div>

          <div className="alert-form-row">
            <label>Важность</label>
            <div className="permission-pills">
              <button type="button" className={severity === "info" ? "active" : ""} onClick={() => setSeverity("info")}>Инфо</button>
              <button type="button" className={severity === "warning" ? "active" : ""} onClick={() => setSeverity("warning")}>Внимание</button>
              <button type="button" className={severity === "critical" ? "active" : ""} onClick={() => setSeverity("critical")}>Критично</button>
            </div>
          </div>

          <div className="alert-form-row">
            <label>Cooldown, мин</label>
            <input type="number" min="0" step="1" value={cooldownMinutes} onChange={(event) => setCooldownMinutes(event.target.value)} />
          </div>
        </div>

        <div className="modal-footer">
          <button type="button" className="btn secondary" onClick={onClose}>Отмена</button>
          <button type="button" className="btn primary" onClick={handleSubmit}>Создать</button>
        </div>
      </div>
    </div>
  );
};
