package admin

import (
	"strings"

	"github.com/GMWalletApp/epusdt/model/dao"
	"github.com/GMWalletApp/epusdt/model/data"
	"github.com/GMWalletApp/epusdt/model/mdb"
	"github.com/GMWalletApp/epusdt/telegram"
	"github.com/labstack/echo/v4"
)

// SettingUpsertItem is a single setting entry for batch upsert.
// Supported groups and keys:
//
//   - group=rate:
//     rate.forced_usdt_rate  (float)  — override USDT/CNY when > 0; <= 0 uses rate.api_url
//     rate.api_url           (string) — external rate API URL used when rate.forced_usdt_rate <= 0
//     rate.adjust_percent    (float)  — rate adjustment percentage
//     rate.okx_c2c_enabled   (bool)   — use OKX C2C rate feed
//
//   - group=epay:
//     epay.default_token     (string) — token for EPAY orders, e.g. "usdt" (default)
//     epay.default_currency  (string) — fiat currency for EPAY orders, e.g. "cny" (default)
//     epay.default_network   (string) — blockchain network for EPAY orders, e.g. "tron" (default)
//
//   - group=okpay:
//     okpay.enabled          (bool)   — enable OkPay as a switch-network payment option
//     okpay.shop_id          (string) — OkPay merchant/shop identifier
//     okpay.shop_token       (string) — OkPay signing token
//     okpay.api_url          (string) — OkPay API base URL
//     okpay.callback_url     (string) — server callback URL used for OkPay notify
//     okpay.return_url       (string) — optional default browser return URL after payment
//     okpay.timeout_seconds  (int)    — outbound OkPay API timeout in seconds
//     okpay.allow_tokens     (string) — comma-separated allowed tokens, e.g. "USDT,TRX"
//
//   - group=brand:
//     brand.checkout_name    (string) — cashier display name (preferred)
//     brand.logo_url         (string) — logo image URL
//     brand.site_title       (string) — payment page title / website title (preferred)
//     brand.success_copy     (string) — text shown on payment success (preferred)
//     brand.support_url      (string) — support / help URL
//     brand.site_name        (string) — legacy cashier/site display name
//     brand.page_title       (string) — legacy payment page title
//     brand.pay_success_text (string) — legacy success text
//
//   - group=system:
//     system.order_expiration_time (int) — order expiry in minutes
type SettingUpsertItem struct {
	Group string `json:"group" enums:"brand,rate,system,epay,okpay" example:"epay"`
	Key   string `json:"key" example:"epay.default_network"`
	Value string `json:"value" example:"tron"`
	Type  string `json:"type" enums:"string,int,bool,json" example:"string"`
}

// SettingsUpsertRequest is the payload for batch upserting settings.
type SettingsUpsertRequest struct {
	Items []SettingUpsertItem `json:"items" validate:"required"`
}

// ListSettings returns all rows, optionally filtered by group.
// @Summary      List settings
// @Description  Returns all settings, optionally filtered by group.
// @Description  Available groups: brand, rate, system, epay, okpay.
// @Description  See SettingUpsertItem for the full list of supported keys per group.
// @Tags         Admin Settings
// @Security     AdminJWT
// @Produce      json
// @Param        group query string false "Group filter (brand|rate|system|epay|okpay)"
// @Success      200 {object} response.ApiResponse{data=[]mdb.Setting}
// @Failure      400 {object} response.ApiResponse
// @Router       /admin/api/v1/settings [get]
func (c *BaseAdminController) ListSettings(ctx echo.Context) error {
	group := strings.ToLower(strings.TrimSpace(ctx.QueryParam("group")))
	rows, err := data.ListSettingsByGroup(group)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, rows)
}

// UpsertSettings batch-inserts / updates rows. Each item is treated
// independently so a malformed row in the middle doesn't drop earlier
// ones. Errors are returned per-key so the UI can surface them.
// @Summary      Upsert settings
// @Description  Batch insert/update settings. Returns per-key status.
// @Description  Supported groups: brand, rate, system, epay, okpay.
// @Description  epay group keys: epay.default_token (e.g. "usdt"), epay.default_currency (e.g. "cny"), epay.default_network (e.g. "tron").
// @Description  okpay group keys: okpay.enabled, okpay.shop_id, okpay.shop_token, okpay.api_url, okpay.callback_url, okpay.return_url, okpay.timeout_seconds, okpay.allow_tokens.
// @Description  rate group keys: rate.forced_usdt_rate (>0 overrides USDT/CNY; <=0 uses rate.api_url), rate.api_url, rate.adjust_percent, rate.okx_c2c_enabled.
// @Description  brand group keys: brand.checkout_name, brand.logo_url, brand.site_title, brand.success_copy, brand.support_url. Legacy aliases brand.site_name, brand.page_title and brand.pay_success_text are also supported.
// @Description  system group keys: system.order_expiration_time.
// @Tags         Admin Settings
// @Security     AdminJWT
// @Accept       json
// @Produce      json
// @Param        request body admin.SettingsUpsertRequest true "Settings payload"
// @Success      200 {object} response.ApiResponse{data=[]admin.SettingsUpsertResult}
// @Failure      400 {object} response.ApiResponse
// @Router       /admin/api/v1/settings [put]
func (c *BaseAdminController) UpsertSettings(ctx echo.Context) error {
	req := new(SettingsUpsertRequest)
	if err := ctx.Bind(req); err != nil {
		return c.FailJson(ctx, err)
	}
	if err := c.ValidateStruct(ctx, req); err != nil {
		return c.FailJson(ctx, err)
	}
	type result struct {
		Key   string `json:"key"`
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}
	out := make([]result, 0, len(req.Items))
	for _, item := range req.Items {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			out = append(out, result{Key: item.Key, OK: false, Error: "key required"})
			continue
		}
		if err := data.SetSetting(item.Group, key, item.Value, item.Type); err != nil {
			out = append(out, result{Key: key, OK: false, Error: err.Error()})
			continue
		}
		out = append(out, result{Key: key, OK: true})
	}

	// When telegram credentials are updated via settings, reload the
	// command bot so operators don't need to restart the process, and
	// sync the notification_channels row so the notify dispatcher picks
	// up the new values immediately.
	telegramKeys := map[string]bool{
		"system.telegram_bot_token":               true,
		"system.telegram_chat_id":                 true,
		"system.telegram_payment_notice_enabled":  true,
		"system.telegram_abnormal_notice_enabled": true,
	}
	for _, item := range req.Items {
		if telegramKeys[strings.TrimSpace(item.Key)] {
			telegram.ReloadBotAsync("settings upsert")
			go dao.SyncTelegramChannelFromSettings()
			break
		}
	}

	return c.SucJson(ctx, out)
}

// DeleteSetting removes one row. The next read of that key will fall
// back to the hardcoded default (see settings_data.GetSetting*).
// @Summary      Delete setting
// @Description  Remove a setting by key (falls back to default)
// @Tags         Admin Settings
// @Security     AdminJWT
// @Produce      json
// @Param        key path string true "Setting key"
// @Success      200 {object} response.ApiResponse
// @Failure      400 {object} response.ApiResponse
// @Router       /admin/api/v1/settings/{key} [delete]
func (c *BaseAdminController) DeleteSetting(ctx echo.Context) error {
	key := strings.TrimSpace(ctx.Param("key"))
	if key == "" {
		return c.SucJson(ctx, nil)
	}
	if err := data.DeleteSetting(key); err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, nil)
}

// Public helper for the rate/usdt overrides — used by config package to
// read settings-backed values without importing the controller package.
var _ = mdb.SettingKeyRateForcedUsdt // ensure key constants remain referenced
