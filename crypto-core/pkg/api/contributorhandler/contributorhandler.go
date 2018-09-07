package contributorhandler

import (
	"net/http"
	"github.com/bfg-dev/crypto-core/pkg/api"
	"github.com/bfg-dev/crypto-core/pkg/services"
	"github.com/pkg/errors"
	"github.com/bfg-dev/crypto-core/pkg/services/contributor"
	"go.uber.org/zap"
	"github.com/bfg-dev/crypto-core/pkg/bfgerrors"
	"github.com/bfg-dev/crypto-core/pkg/templates/contributors"
	"strconv"
	"github.com/bfg-dev/crypto-core/pkg/entities"
	"strings"
	"fmt"
)

type ContributorHandler struct {
	app                services.App
	contributorService contributor.Service
}

func New(
	application services.App,
	contributorService contributor.Service,
) (*ContributorHandler, error) {

	if application == nil {
		return nil, errors.New("ContributorHandler.New, application cannot be empty")
	}

	if contributorService == nil {
		return nil, errors.New("ContributorHandler.New, tokenemission cannot be empty")
	}

	return &ContributorHandler{
		app:                    application,
		contributorService: contributorService,
	}, nil
}

func (h *ContributorHandler) GetNewUserMissionRequests(w http.ResponseWriter, req *http.Request) (*api.Response, error) {

	requests, err := h.contributorService.GetNewMissionRequests()
	if err != nil {
		h.app.Logger().Error("unable to get new missions request", zap.Error(err))
		return nil, bfgerrors.NewApiIntErr(err, nil, "unable to get new missions request", nil)
	}

	return api.SuccessResponse(requests), nil
}

func (h *ContributorHandler) RenderNewUserMissionRequestList(w http.ResponseWriter, req *http.Request) {

	requests, err := h.contributorService.GetNewMissionRequestsList()
	if err != nil {
		h.app.Logger().Error("unable to get new missions request", zap.Error(err))
		return
	}

	for _, request := range requests {
		for key, param := range request.MissionParameters {
			if (!strings.Contains(param, "http")) {
				request.MissionParameters[key] = fmt.Sprint("http://", param)
			}
		}
	}

	p := &contributors.NewRequestsListPage{
		Requests: requests,
	}
	contributors.WritePageTemplate(w, p)

}

func (h *ContributorHandler) SetUserRequestStatus(w http.ResponseWriter, req *http.Request) {

	if err := req.ParseForm(); err != nil {
		w.Write([]byte("cannot parse form"))
		return
	}

	if (req.FormValue("id") == "") {
		w.Write([]byte("id not defines"))
		return
	}

	id, err := strconv.Atoi(req.FormValue("id"))
	if err != nil {
		w.Write([]byte("wrong id format"))
		return
	}

	if (req.FormValue("status") == "") {
		w.Write([]byte("status not defines"))
		return
	}
	status := entities.UserMissionStatus(req.FormValue("status"))

	err = h.contributorService.SetMissionRequestStatus(int64(id), status)
	if err != nil {
		h.app.Logger().Error("unable to set mission request status", zap.Error(err))
		return
	}

	http.Redirect(w, req, "/admin/NewUserMissionRequests", http.StatusFound)
}
