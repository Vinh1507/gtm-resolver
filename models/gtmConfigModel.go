package models

type Domain struct {
	Id          string
	DomainName  string
	Policy      string
	DataCenters []string
	TTL         int
}

type DataCenter struct {
	Id             string
	Domain         string
	Name           string
	IP             string
	Status         string
	HealthCheckUrl string
	Port           int
	Count          int
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

type DataCenterHistory struct {
	DataCenterName string
	HealthCheckUrl string
	Domain         string
	Status         string
	ResponseCode   int
	Reason         string
	TimeStamp      string
}
