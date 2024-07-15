package models

type Domain struct {
	Id          string
	DomainName  string
	DataCenters []DataCenter
	Type        string
}

type DataCenter struct {
	Id             string
	Name           string
	IP             string
	Status         string
	HealthCheckUrl string
	Port           int
	Weight         int
	IsPrimary      bool
	FailoverDelay  int
	FailbackDelay  int
	RankFailover   int
	LoadFeedbacks  []LoadObject
}

type LoadObject struct {
	DataCenterId string
	RelativeUrl  string
	Port         string
	TimeStamp    string
	Tag          string
	Resources    []Resource
}

type Resource struct {
	Name        string
	CurrentLoad int
	TargetLoad  int
	MaxLoad     int
}
