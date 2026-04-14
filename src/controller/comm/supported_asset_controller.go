package comm

import (
	"sort"
	"strconv"

	"github.com/assimon/luuu/model/data"
	"github.com/assimon/luuu/model/response"
	"github.com/assimon/luuu/util/constant"
	"github.com/labstack/echo/v4"
)

type addSupportedAssetRequest struct {
	Network string `json:"network" validate:"required"`
	Token   string `json:"token" validate:"required"`
	Status  int64  `json:"status" validate:"required|in:1,2"`
}

type updateSupportedAssetRequest struct {
	Network string `json:"network" validate:"required"`
	Token   string `json:"token" validate:"required"`
	Status  int64  `json:"status" validate:"required|in:1,2"`
}

// GetSupportedAssets 对外公开可用链与 token 列表（无需鉴权，仅返回已启用项）。
func (c *BaseCommController) GetSupportedAssets(ctx echo.Context) error {
	list, err := data.ListEnabledSupportedAssets()
	if err != nil {
		return c.FailJson(ctx, err)
	}
	wallets, err := data.GetAvailableWalletAddress()
	if err != nil {
		return c.FailJson(ctx, err)
	}

	networkSet := make(map[string]struct{})
	for _, w := range wallets {
		networkSet[w.Network] = struct{}{}
	}

	grouped := make(map[string][]string)
	for _, item := range list {
		if _, ok := networkSet[item.Network]; !ok {
			continue
		}
		grouped[item.Network] = append(grouped[item.Network], item.Token)
	}

	networks := make([]string, 0, len(grouped))
	for network := range grouped {
		networks = append(networks, network)
	}
	sort.Strings(networks)

	supports := make([]response.NetworkTokenSupport, 0, len(networks))
	for _, network := range networks {
		tokens := grouped[network]
		sort.Strings(tokens)
		supports = append(supports, response.NetworkTokenSupport{
			Network: network,
			Tokens:  tokens,
		})
	}

	return c.SucJson(ctx, response.SupportedAssetsResponse{Supports: supports})
}

// ListSupportedAssetRecords 查询支持项明细（无需鉴权）。
func (c *BaseCommController) ListSupportedAssetRecords(ctx echo.Context) error {
	network := ctx.QueryParam("network")
	list, err := data.ListSupportedAssets(network)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, list)
}

// GetSupportedAsset 查询单条支持项（无需鉴权）。
func (c *BaseCommController) GetSupportedAsset(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		return c.FailJson(ctx, constant.ParamsMarshalErr)
	}
	asset, err := data.GetSupportedAssetByID(id)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	if asset.ID <= 0 {
		return c.FailJson(ctx, constant.SupportedAssetNotFound)
	}
	return c.SucJson(ctx, asset)
}

// AddSupportedAsset 新增支持项（鉴权）。
func (c *BaseCommController) AddSupportedAsset(ctx echo.Context) error {
	req := new(addSupportedAssetRequest)
	if err := ctx.Bind(req); err != nil {
		return c.FailJson(ctx, constant.ParamsMarshalErr)
	}
	if err := c.ValidateStruct(ctx, req); err != nil {
		return c.FailJson(ctx, err)
	}
	asset, err := data.AddSupportedAsset(req.Network, req.Token, req.Status)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, asset)
}

// UpdateSupportedAsset 修改支持项（鉴权）。
func (c *BaseCommController) UpdateSupportedAsset(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		return c.FailJson(ctx, constant.ParamsMarshalErr)
	}
	req := new(updateSupportedAssetRequest)
	if err := ctx.Bind(req); err != nil {
		return c.FailJson(ctx, constant.ParamsMarshalErr)
	}
	if err := c.ValidateStruct(ctx, req); err != nil {
		return c.FailJson(ctx, err)
	}
	asset, err := data.UpdateSupportedAsset(id, req.Network, req.Token, req.Status)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, asset)
}

// DeleteSupportedAsset 删除支持项（鉴权）。
func (c *BaseCommController) DeleteSupportedAsset(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		return c.FailJson(ctx, constant.ParamsMarshalErr)
	}
	asset, err := data.GetSupportedAssetByID(id)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	if asset.ID <= 0 {
		return c.FailJson(ctx, constant.SupportedAssetNotFound)
	}
	if err := data.DeleteSupportedAssetByID(id); err != nil {
		return c.FailJson(ctx, err)
	}
	return c.SucJson(ctx, nil)
}
