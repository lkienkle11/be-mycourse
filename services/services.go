package services

import (
	"mycourse-io-be/models"
	"mycourse-io-be/repository"
)

type Service struct {
	Repo *repository.Repository
}

func New() *Service {
	return &Service{
		Repo: repository.New(models.DB),
	}
}
