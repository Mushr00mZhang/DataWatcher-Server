package services

import "server/modules"

type ElasticService struct {
	Elastic *modules.Elastic
}

func NewElasticService(elastic *modules.Elastic) *ElasticService {
	return &ElasticService{
		Elastic: elastic,
	}
}
