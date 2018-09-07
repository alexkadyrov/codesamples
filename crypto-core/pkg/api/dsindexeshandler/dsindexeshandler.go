package dsindexeshandler

import (
	"net/http"
	"strings"

	"github.com/bfg-dev/crypto-core/pkg/api"
	"github.com/bfg-dev/crypto-core/pkg/api/params"
	"github.com/bfg-dev/crypto-core/pkg/api/params/dsindexes"
	"github.com/bfg-dev/crypto-core/pkg/bfgerrors"
	"github.com/bfg-dev/crypto-core/pkg/helpers/currency"
	"github.com/bfg-dev/crypto-core/pkg/services"
	"github.com/bfg-dev/crypto-core/pkg/services/asset"
	"github.com/bfg-dev/crypto-core/pkg/services/blockchain"
	"github.com/bfg-dev/crypto-core/pkg/services/cryptofund"
	"github.com/bfg-dev/crypto-core/pkg/services/tokenemission"
	"github.com/bfg-dev/crypto-core/pkg/services/tokenredemption"
	"github.com/bfg-dev/crypto-core/pkg/types"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type DSIndexesHandler struct {
	app                    services.App
	assetService           asset.Service
	tokenEmissionService   tokenemission.Service
	tokenRedemptionService tokenredemption.Service
	buybackService         blockchain.BuybackService
	cryptofundService      cryptofund.Service
}

func New(
	application services.App,
	assetsrv asset.Service,
	tokenemissionsrv tokenemission.Service,
	tokenredemptionsrv tokenredemption.Service,
	buybackService blockchain.BuybackService,
	cryptofundsrv cryptofund.Service,
) (*DSIndexesHandler, error) {

	if application == nil {
		return nil, errors.New("assethandler.New, application must be not empty")
	}

	if tokenemissionsrv == nil {
		return nil, errors.New("DSIndexesHandler.New, tokenemission must be not empty")
	}

	if assetsrv == nil {
		return nil, errors.New("DSIndexesHandler.New, assetsrv must be not empty")
	}

	if tokenredemptionsrv == nil {
		return nil, errors.New("DSIndexesHandler.New, tokenredemptionsrv must be not empty")
	}

	if buybackService == nil {
		return nil, errors.New("DSIndexesHandler.New, buybackService must be not empty")
	}

	if cryptofundsrv == nil {
		return nil, errors.New("DSIndexesHandler.New, cryptofundsrv must be not empty")
	}

	return &DSIndexesHandler{
		app:                    application,
		assetService:           assetsrv,
		tokenEmissionService:   tokenemissionsrv,
		tokenRedemptionService: tokenredemptionsrv,
		buybackService:         buybackService,
		cryptofundService:      cryptofundsrv,
	}, nil
}

func (h *DSIndexesHandler) GetSummary(w http.ResponseWriter, req *http.Request) (*api.Response, error) {
	methodParams := dsindexes.NewSummaryParams(*req.URL)
	if validationErrors := params.MustValidateParams(&methodParams); validationErrors != nil {
		return api.ErrorResponse(validationErrors), nil
	}

	assetSymbol := strings.ToLower(methodParams.AssetSymbol[3:])

	asset, err := h.assetService.GetAssetBySymbol(assetSymbol)
	if err != nil || asset == nil {
		h.app.Logger().Error("unable to get asset by symbol", zap.String("assetSymbol", assetSymbol), zap.Error(err))
		return api.ErrorResponse("unable to get asset by symbol"), nil
	}

	// get ledger by asset
	ledgerIDs, err := h.tokenEmissionService.GetLedgerIDs([]int64{asset.ID})
	if err != nil {
		h.app.Logger().Error("unable to get ledger ids", zap.Int64("asset.ID", asset.ID), zap.Error(err))
		return nil, bfgerrors.NewApiIntErr(err, methodParams, "unable to get ledger ids", nil)
	}

	issuedTokenCount, err := h.tokenEmissionService.GetIssuedTokenCount(ledgerIDs)
	if err != nil {
		h.app.Logger().Error("unable to get issuedTokenCount", zap.Int64s("ledgerIDs", ledgerIDs), zap.Error(err))
		return nil, bfgerrors.NewApiIntErr(err, nil, "unable to get issuedTokenCount", types.MapI{"url": req.URL})
	}

	tobeIssuedTokenCount, err := h.tokenEmissionService.GetNotIssuedTokenCount(ledgerIDs)
	if err != nil {
		h.app.Logger().Error("unable to get GetNotIssuedTokenCount", zap.Int64s("ledgerIDs", ledgerIDs), zap.Error(err))
		return nil, bfgerrors.NewApiIntErr(err, nil, "unable to get GetNotIssuedTokenCount", types.MapI{"url": req.URL})
	}

	burnedBuyback, err := h.buybackService.GetBuybackBurnedAmount(asset.ID)
	if err != nil {
		h.app.Logger().Error("GetBuybackBurnedAmount error", zap.Int64("assetID", asset.ID), zap.Error(err))
		return nil, bfgerrors.NewApiIntErr(err, nil, "GetBuybackBurnedAmount error", nil)
	}

	// burned redemption
	redemptionBooks, err := h.tokenRedemptionService.GetActiveBooks([]int64{asset.ID})
	if err != nil {
		h.app.Logger().Error("Cannot get active burningman books by asset id", zap.Int64("asset.ID", asset.ID), zap.Error(err))
		return nil, bfgerrors.NewApiIntErr(err, nil, "Cannot get active burningman books by asset id", nil)
	}

	redemptionBookIDs := make([]int64, len(redemptionBooks))
	for i, item := range redemptionBooks {
		redemptionBookIDs[i] = item.ID
	}

	burnedRedemption, err := h.tokenRedemptionService.GetRedemptionBurnedTotalAmount(redemptionBookIDs)
	if err != nil {
		h.app.Logger().Error("Cannot get GetRedemptionBurnedTotalAmount", zap.Int64s("redemptionBookIDs", redemptionBookIDs), zap.Error(err))
		return nil, bfgerrors.NewApiIntErr(err, nil, "Cannot get GetRedemptionBurnedTotalAmount", nil)
	}

	tobeBurnedTokenCount, err := h.tokenRedemptionService.GetRedemptionTobeBurnedTotalAmount(redemptionBookIDs)
	if err != nil {
		h.app.Logger().Error("unable to get GetRedemptionTobeBurnedTotalAmount", zap.Int64s("redemptionBookIDs", redemptionBookIDs), zap.Error(err))
		return nil, bfgerrors.NewApiIntErr(err, nil, "unable to get GetRedemptionTobeBurnedTotalAmount", types.MapI{"url": req.URL})
	}

	data := map[string]decimal.Decimal{
		"issued":            currency.DenormalizeATx(*issuedTokenCount),
		"to_be_issued":      currency.DenormalizeATx(*tobeIssuedTokenCount),
		"to_be_burned":      currency.DenormalizeATx(*tobeBurnedTokenCount),
		"burned_buyback":    currency.DenormalizeATx(*burnedBuyback),
		"burned_redemption": currency.DenormalizeATx((*burnedRedemption).Add(*burnedBuyback)),
	}

	return api.SuccessResponse(data), nil
}

func (h *DSIndexesHandler) GetSummaryV2(w http.ResponseWriter, req *http.Request) (*api.Response, error) {
	methodParams := dsindexes.NewSummaryParams(*req.URL)
	if validationErrors := params.MustValidateParams(&methodParams); validationErrors != nil {
		return api.ErrorResponse(validationErrors), nil
	}

	assetSymbol := strings.ToLower(methodParams.AssetSymbol[3:])

	a, err := h.assetService.GetAssetBySymbol(assetSymbol)
	if err != nil || a == nil {
		h.app.Logger().Error("unable to get asset by symbol", zap.String("assetSymbol", assetSymbol), zap.Error(err))
		return api.ErrorResponse("unable to get asset by symbol"), nil
	}

	ledgerIDs, err := h.tokenEmissionService.GetLedgerIDs([]int64{a.ID})
	if err != nil {
		h.app.Logger().Error("unable to get ledger ids", zap.Int64("asset.ID", a.ID), zap.Error(err))
		return nil, bfgerrors.NewApiIntErr(err, methodParams, "unable to get ledger ids", nil)
	}

	issuedTokenCount, err := h.tokenEmissionService.GetIssuedTokenCount(ledgerIDs)
	if err != nil {
		h.app.Logger().Error("unable to get issuedTokenCount", zap.Int64s("ledgerIDs", ledgerIDs), zap.Error(err))
		return nil, bfgerrors.NewApiIntErr(err, nil, "unable to get issuedTokenCount", types.MapI{"url": req.URL})
	}

	tobeIssuedTokenCount, err := h.tokenEmissionService.GetNotIssuedTokenCount(ledgerIDs)
	if err != nil {
		h.app.Logger().Error("unable to get GetNotIssuedTokenCount", zap.Int64s("ledgerIDs", ledgerIDs), zap.Error(err))
		return nil, bfgerrors.NewApiIntErr(err, nil, "unable to get GetNotIssuedTokenCount", types.MapI{"url": req.URL})
	}

	// burned on buybacks
	burnedBuyback, err := h.buybackService.GetBuybackBurnedAmount(a.ID)
	if err != nil {
		h.app.Logger().Error("GetBuybackBurnedAmount error", zap.Int64("assetID", a.ID), zap.Error(err))
		return nil, bfgerrors.NewApiIntErr(err, nil, "GetBuybackBurnedAmount error", nil)
	}

	// burned on burningman
	redemptionBooks, err := h.tokenRedemptionService.GetActiveBooks([]int64{a.ID})
	if err != nil {
		h.app.Logger().Error("Cannot get active burningman books by asset id", zap.Int64("asset.ID", a.ID), zap.Error(err))
		return nil, bfgerrors.NewApiIntErr(err, nil, "Cannot get active burningman books by asset id", nil)
	}

	redemptionBookIDs := make([]int64, len(redemptionBooks))
	for i, item := range redemptionBooks {
		redemptionBookIDs[i] = item.ID
	}

	burnedBurningman, err := h.tokenRedemptionService.GetRedemptionBurnedTotalAmount(redemptionBookIDs)
	if err != nil {
		h.app.Logger().Error("Cannot get GetRedemptionBurnedTotalAmount", zap.Int64s("redemptionBookIDs", redemptionBookIDs), zap.Error(err))
		return nil, bfgerrors.NewApiIntErr(err, nil, "Cannot get GetRedemptionBurnedTotalAmount", nil)
	}

	tobeBurnedTokenCount, err := h.tokenRedemptionService.GetRedemptionTobeBurnedTotalAmount(redemptionBookIDs)
	if err != nil {
		h.app.Logger().Error("unable to get GetRedemptionTobeBurnedTotalAmount", zap.Int64s("redemptionBookIDs", redemptionBookIDs), zap.Error(err))
		return nil, bfgerrors.NewApiIntErr(err, nil, "unable to get GetRedemptionTobeBurnedTotalAmount", types.MapI{"url": req.URL})
	}

	burnedTotal := burnedBurningman.Add(*burnedBuyback)

	data := map[string]decimal.Decimal{
		"total_issued":      currency.DenormalizeATx(*issuedTokenCount),
		"total_supply":      currency.DenormalizeATx((*issuedTokenCount).Sub(burnedTotal)),
		"to_be_issued":      currency.DenormalizeATx(*tobeIssuedTokenCount),
		"to_be_burned":      currency.DenormalizeATx(*tobeBurnedTokenCount),
		"burned_buyback":    currency.DenormalizeATx(*burnedBuyback),
		"burned_redemption": currency.DenormalizeATx(*burnedBurningman),
		"burned_total":      currency.DenormalizeATx(burnedTotal),
	}

	return api.SuccessResponse(data), nil
}
