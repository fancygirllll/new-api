package dto

type UserSetting struct {
	NotifyType                       string  `json:"notify_type,omitempty"`                          // QuotaWarningType is the quota warning type
	QuotaWarningThreshold            float64 `json:"quota_warning_threshold,omitempty"`              // QuotaWarningThreshold is the quota warning threshold
	WebhookUrl                       string  `json:"webhook_url,omitempty"`                          // WebhookUrl is the webhook address
	WebhookSecret                    string  `json:"webhook_secret,omitempty"`                       // WebhookSecret is the webhook secret
	NotificationEmail                string  `json:"notification_email,omitempty"`                   // NotificationEmail is the notification email address
	BarkUrl                          string  `json:"bark_url,omitempty"`                             // BarkUrl is the Bark push URL
	GotifyUrl                        string  `json:"gotify_url,omitempty"`                           // GotifyUrl is the Gotify server address
	GotifyToken                      string  `json:"gotify_token,omitempty"`                         // GotifyToken is the Gotify application token
	GotifyPriority                   int     `json:"gotify_priority"`                                // GotifyPriority is the Gotify message priority
	UpstreamModelUpdateNotifyEnabled bool    `json:"upstream_model_update_notify_enabled,omitempty"` // Whether to receive scheduled upstream model update notifications (admins only)
	AcceptUnsetRatioModel            bool    `json:"accept_unset_model_ratio_model,omitempty"`       // AcceptUnsetRatioModel controls whether models without a configured price are accepted
	RecordIpLog                      bool    `json:"record_ip_log,omitempty"`                        // Whether to record IPs in request and error logs
	SidebarModules                   string  `json:"sidebar_modules,omitempty"`                      // SidebarModules is the left sidebar module configuration
	BillingPreference                string  `json:"billing_preference,omitempty"`                   // BillingPreference is the billing strategy (subscription/wallet)
	Language                         string  `json:"language,omitempty"`                             // Language is the user language preference (zh, en)
}

var (
	NotifyTypeEmail   = "email"   // Email
	NotifyTypeWebhook = "webhook" // Webhook
	NotifyTypeBark    = "bark"    // Bark push
	NotifyTypeGotify  = "gotify"  // Gotify push
)
