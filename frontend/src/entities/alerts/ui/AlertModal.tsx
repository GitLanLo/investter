import React, { useState } from "react";
import { X, Bell, Plus, Minus, Info, AlertTriangle, AlertCircle } from "lucide-react";

interface AlertModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (rule: any) => void;
  ticker: string;
}

export const AlertModal: React.FC<AlertModalProps> = ({ isOpen, onClose, onSubmit, ticker }) => {
  const [eventType, setEventType] = useState("price");
  const [targetIndicator, setTargetIndicator] = useState("");
  const [operator, setOperator] = useState(">");
  const [threshold, setThreshold] = useState("");
  const [secondaryThreshold, setSecondaryThreshold] = useState("");
  const [severity, setSeverity] = useState("info");
  const [direction, setDirection] = useState("up");
  const [expiresAt, setExpiresAt] = useState("");
  const [triggerMode, setTriggerMode] = useState("once");
  const [channels, setChannels] = useState(["app"]);

  if (!isOpen) return null;

  const handleSubmit = () => {
    let finalEventType = eventType;
    if (eventType === "signal") {
      finalEventType = "decision_threshold_triggered";
    }

    onSubmit({
      ticker,
      event_type: finalEventType,
      target_indicator: targetIndicator,
      operator,
      threshold: parseFloat(threshold) || 0,
      secondary_threshold: parseFloat(secondaryThreshold) || 0,
      severity,
      direction: (finalEventType === "decision_threshold_triggered" || finalEventType === "signal") ? direction : "",
      expires_at: expiresAt ? new Date(expiresAt).toISOString() : null,
      trigger_mode: triggerMode,
      delivery_channels: channels,
      is_enabled: true,
      cooldown_minutes: 60,
    });
  };

  const toggleChannel = (ch: string) => {
    setChannels(prev => prev.includes(ch) ? prev.filter(c => c !== ch) : [...prev, ch]);
  };

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal-content advanced-alert-modal" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <h2><Bell size={20} /> Создать уведомление для <strong>{ticker}</strong></h2>
          <button type="button" className="modal-close" onClick={onClose}><X size={20} /></button>
        </div>
        
        <div className="modal-body">
          <div className="alert-form-row">
            <label>Источник данных</label>
            <select value={eventType} onChange={e => setEventType(e.target.value)}>
              <option value="price">Рыночная цена</option>
              <option value="indicator">Тех. индикатор</option>
              <option value="signal">ML Прогноз</option>
              <option value="volume">Объем торгов</option>
            </select>
          </div>

          {eventType === "indicator" && (
            <div className="alert-form-row">
              <label>Индикатор</label>
              <select value={targetIndicator} onChange={e => setTargetIndicator(e.target.value)}>
                <option value="">Выберите...</option>
                <option value="RSI">RSI (Relative Strength Index)</option>
                <option value="EMA">EMA (Moving Average)</option>
                <option value="MACD">MACD</option>
                <option value="ATR">ATR</option>
              </select>
            </div>
          )}

          <div className="alert-form-row">
            <label>Условие</label>
            <div className="alert-controls-group">
              <div className="alert-controls-row">
                <select value={operator} onChange={e => setOperator(e.target.value)}>
                  <option value=">">Больше {">"}</option>
                  <option value="<">Меньше {"<"}</option>
                  <option value="cross_up">Пересечение вверх</option>
                  <option value="cross_down">Пересечение вниз</option>
                  <option value="inside_channel">Внутри канала</option>
                  <option value="outside_channel">Вне канала</option>
                  <option value="move_up_percent">Рост %</option>
                  <option value="move_down_percent">Снижение %</option>
                </select>
                
                {eventType === "signal" ? (
                  <select value={direction} onChange={e => setDirection(e.target.value)}>
                    <option value="up">Рост</option>
                    <option value="down">Снижение</option>
                  </select>
                ) : (
                  <input 
                    type="number" 
                    placeholder="Значение" 
                    value={threshold} 
                    onChange={e => setThreshold(e.target.value)} 
                  />
                )}
              </div>
              
              {(operator === "inside_channel" || operator === "outside_channel") && (
                <div className="alert-controls-row">
                  <input 
                    type="number" 
                    placeholder="Второе значение (граница)" 
                    value={secondaryThreshold} 
                    onChange={e => setSecondaryThreshold(e.target.value)} 
                  />
                </div>
              )}
            </div>
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
            <label>Срок действия</label>
            <input type="datetime-local" value={expiresAt} onChange={e => setExpiresAt(e.target.value)} />
          </div>

          <div className="alert-form-row">
            <label>Срабатывание</label>
            <select value={triggerMode} onChange={e => setTriggerMode(e.target.value)}>
              <option value="once">Один раз</option>
              <option value="always">Каждый раз</option>
              <option value="once_per_day">Один раз в день</option>
            </select>
          </div>

          <div className="alert-form-row">
            <label>Каналы связи</label>
            <div className="permission-pills">
              <button type="button" className={channels.includes("app") ? "active" : ""} onClick={() => toggleChannel("app")}>Приложение</button>
              <button type="button" className={channels.includes("telegram") ? "active" : ""} onClick={() => toggleChannel("telegram")}>Telegram</button>
              <button type="button" className={channels.includes("email") ? "active" : ""} onClick={() => toggleChannel("email")}>Email</button>
            </div>
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
