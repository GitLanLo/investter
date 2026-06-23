import React, { useEffect, useRef } from "react";
import { api } from "../shared/api/client";
import { useToast, ToastSeverity } from "../shared/ui/Toast";

interface NotificationEvent {
  id: number;
  event_type: string;
  message: string;
  severity: string;
  ticker?: string;
  created_at: string;
}

export const NotificationManager: React.FC = () => {
  const { showToast } = useToast();
  // Using a ref to track the highest ID seen in this session
  const maxSeenIdRef = useRef<number | null>(null);

  useEffect(() => {
    let timeoutId: NodeJS.Timeout;

    const fetchEvents = async () => {
      try {
        const response = await api.get<{ items: NotificationEvent[] }>("/api/v1/alerts/events?limit=10");
        const events = response.items || [];
        
        if (events.length === 0) return;

        // If this is the very first fetch in the session, just initialize the max ID
        // so we don't spam the user with old events immediately on load.
        if (maxSeenIdRef.current === null) {
          const maxId = Math.max(...events.map(e => e.id));
          maxSeenIdRef.current = maxId;
          return;
        }

        // Filter events that have an ID greater than the max seen ID
        const newEvents = events.filter(e => e.id > maxSeenIdRef.current!);
        
        if (newEvents.length > 0) {
          // Update the max ID to prevent re-showing
          const newMaxId = Math.max(...newEvents.map(e => e.id));
          maxSeenIdRef.current = newMaxId;

          // Show toasts for the new events
          // Reverse them so they appear in chronological order (oldest new first)
          newEvents.reverse().forEach(event => {
            let severity: ToastSeverity = "info";
            if (event.severity === "warning") severity = "warning";
            if (event.severity === "critical") severity = "critical";

            const title = event.ticker ? `Событие по ${event.ticker}` : "Системное уведомление";

            showToast({
              title,
              message: event.message,
              severity,
            });
          });
        }
      } catch (err) {
        console.error("Failed to fetch notification events:", err);
      } finally {
        timeoutId = setTimeout(fetchEvents, 10000); // Poll every 10 seconds
      }
    };

    fetchEvents();

    return () => {
      clearTimeout(timeoutId);
    };
  }, [showToast]);

  return null; // This is a logic-only component
};
