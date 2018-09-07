package postgres

import (
	"github.com/bfg-dev/crypto-core/pkg/entities"
	"github.com/bfg-dev/crypto-core/pkg/helpers/db"
	"github.com/bfg-dev/crypto-core/pkg/services/contributor"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type missionRepository struct {
	db sqlx.Ext
}

func (repo *missionRepository) GetByID(id int64) (*entities.CCMission, error) {
	rs := entities.CCMission{}
	row := repo.db.QueryRowx(`SELECT * FROM "ccMissions" WHERE "id" = $1`, id)

	err := row.StructScan(&rs)
	if err != nil {
		return nil, db.EmptyOrError(err, "missionRepository.GetByID, unable to get mission by id")
	}

	return &rs, nil
}


func NewMissionRepository(db *sqlx.DB) (contributor.MissionRepository, error) {
	if db == nil {
		return nil, errors.New("NewRoleRepository: db connection is empty")
	}

	return &missionRepository{db}, nil
}
