package contributor

import (
	"github.com/bfg-dev/crypto-core/pkg/entities"
	"time"
)

type UserMissionRequest struct {
	ID                       int64
	CreatedAt                time.Time
	UserName                 string
	Mission                  string
	MissionParameters        map[string]string
}

type MissionRepository interface {
	GetByID(id int64) (*entities.CCMission, error)
}

type UserMissionRepository interface {
	GetByID(id int64) (*entities.CCUserMission, error)
	GetNewMissionRequests() ([]entities.CCUserMission, error)
	GetNewMissionRequestsList() ([]UserMissionRequest, error)
	SetMissionRequestStatus(id int64, status entities.UserMissionStatus) error
}

type Service interface {
	GetNewMissionRequests() ([]entities.CCUserMission, error)
	GetNewMissionRequestsList() ([]UserMissionRequest, error)
	SetMissionRequestStatus(id int64, status entities.UserMissionStatus) error
}

