package contributor

import (
	"github.com/bfg-dev/crypto-core/pkg/entities"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type service struct {
	db              *sqlx.DB
	missionRepo     MissionRepository
	userMissionRepo UserMissionRepository
}

func (s *service) GetNewMissionRequests() ([]entities.CCUserMission, error) {
	requests, err := s.userMissionRepo.GetNewMissionRequests()
	if err != nil {
		return nil, errors.Wrap(err, "contributor.GetNewMissionRequests, unable to get new requests")
	}
	return requests, nil
}

func (s *service) GetNewMissionRequestsList() ([]UserMissionRequest, error) {
	requests, err := s.userMissionRepo.GetNewMissionRequestsList()
	if err != nil {
		return nil, errors.Wrap(err, "contributor.GetNewMissionRequests, unable to get new requests")
	}
	return requests, nil
}

func (s *service) SetMissionRequestStatus(id int64, status entities.UserMissionStatus) error {
	err := s.userMissionRepo.SetMissionRequestStatus(id, status)
	if err != nil {
		return errors.Wrap(err, "contributor.SetMissionRequestStatus, unable to set request status")
	}
	return nil
}

func NewService(
	missionRepo MissionRepository,
	userMissionRepo UserMissionRepository,
	) (Service, error) {

	if missionRepo == nil {
		return nil, errors.New("contributor.NewService, missionRepo cannot be empty")
	}

	if userMissionRepo == nil {
		return nil, errors.New("contributor.NewService, userMissionRepo cannot be empty")
	}

	return &service{
		missionRepo:     missionRepo,
		userMissionRepo: userMissionRepo,
	}, nil
}
