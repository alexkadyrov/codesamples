package postgres

import (
	"github.com/bfg-dev/crypto-core/pkg/entities"
	"github.com/bfg-dev/crypto-core/pkg/helpers/db"
	"github.com/bfg-dev/crypto-core/pkg/services/contributor"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"encoding/json"
)

type userMissionRepository struct {
	db sqlx.Ext
}

func (repo *userMissionRepository) GetByID(id int64) (*entities.CCUserMission, error) {
	rs := entities.CCUserMission{}
	row := repo.db.QueryRowx(`SELECT * FROM "ccUserMissions" WHERE "id" = $1`, id)

	err := row.StructScan(&rs)
	if err != nil {
		return nil, db.EmptyOrError(err, "missionRepository.GetByID, unable to get mission by id")
	}

	return &rs, nil
}

func (repo *userMissionRepository) GetNewMissionRequests() ([]entities.CCUserMission, error) {

	rows, err := repo.db.Queryx(`SELECT * FROM "ccUserMissions" WHERE "status" = 'new'`)

	if err != nil {
		return nil, db.EmptyOrError(err, "missionRepository.GetNewMissionRequests, unable to get list")
	}

	userMissions := make([]entities.CCUserMission, 0)
	userMission := entities.CCUserMission{}

	for rows.Next() {
		err = rows.StructScan(&userMission)

		if err != nil {
			return nil, errors.Wrap(err, "missionRepository.GetNewMissionRequests, unable to scan misison to struct")
		}

		userMissions = append(userMissions, userMission)
	}

	return userMissions, nil
}

func (repo *userMissionRepository) GetNewMissionRequestsList() ([]contributor.UserMissionRequest, error) {

	rows, err := repo.db.Queryx(
		`SELECT
			us.ID, us."createdAt", us."missionParameters",
  			u.firstname, u.lastname,
			m.title AS "Mission"
		FROM
   			"ccUserMissions" us
   		JOIN
   			"users" u ON us."userId" = u.id
   		JOIN
   			"ccMissions" m ON us."missionId" = m.id
		WHERE
			us."status" = 'new'`)

	if err != nil {
		return nil, db.EmptyOrError(err, "missionRepository.GetNewMissionRequests, unable to get list")
	}

	userMissions := make([]contributor.UserMissionRequest, 0)
	var firstName string
	var lastName string
	var missionParameters *string

	for rows.Next() {
		userMission := contributor.UserMissionRequest{}
		err = rows.Scan(
			&userMission.ID,
			&userMission.CreatedAt,
			&missionParameters,
			&firstName,
			&lastName,
			&userMission.Mission,
		)

		if err != nil {
			return nil, errors.Wrap(err, "missionRepository.GetNewMissionRequests, unable to scan misison to struct")
		}

		json.Unmarshal([]byte(*missionParameters), &userMission.MissionParameters)

		//todo сделать обработку, если нет firstName, либо нет lastName, либо ни того ни другого (взять email)
		userMission.UserName = firstName + " " + lastName
		userMissions = append(userMissions, userMission)
	}

	return userMissions, nil
}

func (repo *userMissionRepository) SetMissionRequestStatus(id int64, status entities.UserMissionStatus) error {
	_, err := repo.db.Exec(`
		UPDATE
			"ccUserMissions"
		SET
			status=$2
		WHERE
			id=$1
	`, id, status)

	return err
}

func NewUserMissionRepository(db *sqlx.DB) (contributor.UserMissionRepository, error) {
	if db == nil {
		return nil, errors.New("NewUserMissionRepository: db connection is empty")
	}

	return &userMissionRepository{db}, nil
}
