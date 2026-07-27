# TODOS


### Betrieb & Resilienz (Operations)

• Hot-Reloading der Konfiguration: Aktuell erfordert eine Änderung an der config.json
einen Neustart des Daemons. Durch das Abfangen eines SIGHUP -Signals oder den Einsatz von
File-Watchern (wie fsnotify ) könnte der Scheduler neue Konfigurationen on-the-fly einlesen.
• Proaktives Alerting: Bislang landen fehlerhafte Jobs in der Dead Letter Queue (DLQ) oder
im Audit-Log. Der Scheduler könnte um ein Notification-Modul erweitert werden (z. B. E-Mail,
Slack/Teams Webhooks), das Alarm schlägt, wenn ein Job x-mal hintereinander fehlschlägt oder
die DLQ-Einträge einen kritischen Schwellenwert überschreiten.

### Monitoring & Observability

• Prometheus Metrics Exporter: sind (noch) nicht vollständig umgesetzt, sollten hier detaillierte Go-Metriken
(Routinen, Memory) sowie Business-Metriken (Anzahl gestarteter Jobs, Success/Failure Rates,
DLQ-Größe) exponiert werden.
• Verteilte Tracing-IDs: Da der Scheduler Collector-, Transformation- und Delivery-Layer
orchestriert, könnte jedem Lauf (Run) eine eindeutige Correlation-ID / Trace-ID mitgegeben
werden, die an alle aufgerufenen Binaries/Systeme durchgereicht wird. So lassen sich Fehler
über Log-Grenzen hinweg perfekt nachvollziehen.

---

