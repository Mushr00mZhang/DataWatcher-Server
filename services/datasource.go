package services

import "server/modules"

type DatasourceService struct {
	Datasources *[]*modules.Datasource
}

func NewDatasourceService(datasources *[]*modules.Datasource) *DatasourceService {
	return &DatasourceService{
		Datasources: datasources,
	}
}
func (service DatasourceService) GetDatasources() []string {
	list := make([]string, len(*service.Datasources))
	for count, i := len(*service.Datasources), 0; i < count; i++ {
		list[i] = (*service.Datasources)[i].Code
	}
	return list
}
func (service DatasourceService) GetDatasource(code string) (*modules.Datasource, error) {
	for _, datasource := range *service.Datasources {
		if datasource.Code == code {
			return datasource, nil
		}
	}
	return nil, modules.ErrDatasourceNotFound
}
