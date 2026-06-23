import React, { createContext, useContext, useState, useCallback } from "react";
import { AlertCircle, AlertTriangle, Info, X } from "lucide-react";

export type ToastSeverity = "info" | "warning" | "critical";

export interface ToastMessage {
  id: string;
  title: string;
  message?: string;
  severity: ToastSeverity;
}

interface ToastContextValue {
  showToast: (toast: Omit<ToastMessage, "id">) => void;
  removeToast: (id: string) => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

export function useToast() {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error("useToast must be used within a ToastProvider");
  }
  return context;
}

export const ToastProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [toasts, setToasts] = useState<ToastMessage[]>([]);

  const showToast = useCallback((toast: Omit<ToastMessage, "id">) => {
    const id = Math.random().toString(36).substring(2, 9);
    setToasts((current) => [...current, { ...toast, id }]);
    
    // Auto-remove after 6 seconds
    setTimeout(() => {
      removeToast(id);
    }, 6000);
  }, []);

  const removeToast = useCallback((id: string) => {
    setToasts((current) => current.filter((t) => t.id !== id));
  }, []);

  return (
    <ToastContext.Provider value={{ showToast, removeToast }}>
      {children}
      <div className="toast-container">
        {toasts.map((toast) => (
          <div key={toast.id} className={`toast toast-${toast.severity}`}>
            <div className="toast-icon">
              {toast.severity === "info" && <Info size={20} />}
              {toast.severity === "warning" && <AlertTriangle size={20} />}
              {toast.severity === "critical" && <AlertCircle size={20} />}
            </div>
            <div className="toast-content">
              <strong>{toast.title}</strong>
              {toast.message && <p>{toast.message}</p>}
            </div>
            <button type="button" className="toast-close" onClick={() => removeToast(toast.id)}>
              <X size={16} />
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
};
