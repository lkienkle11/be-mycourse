package services

import "mycourse-io-be/models"

type Service struct {
	Repo *models.Repository
}

func New() *Service {
	return &Service{
		Repo: models.NewRepository(),
	}
}
