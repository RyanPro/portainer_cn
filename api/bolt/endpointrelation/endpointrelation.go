package endpointrelation

import (
	"github.com/boltdb/bolt"
	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/bolt/internal"
)

const (
	// BucketName represents the name of the bucket where this service stores data.
	BucketName = "endpoint_relations"
)

// Service represents a service for managing endpoint relation data.
type Service struct {
	db *bolt.DB
}

// NewService creates a new instance of a service.
func NewService(db *bolt.DB) (*Service, error) {
	err := internal.CreateBucket(db, BucketName)
	if err != nil {
		return nil, err
	}

	return &Service{
		db: db,
	}, nil
}

// EndpointRelation returns a Endpoint relation object by EndpointID
func (service *Service) EndpointRelation(endpointID portainer.EndpointID) (*portainer.EndpointRelation, error) {
	var endpointRelation portainer.EndpointRelation
	identifier := internal.Itob(int(endpointID))

	err := internal.GetObject(service.db, BucketName, identifier, &endpointRelation)
	if err != nil {
		return nil, err
	}

	return &endpointRelation, nil
}

// CreateEndpointRelation saves endpointRelation
func (service *Service) CreateEndpointRelation(endpointRelation *portainer.EndpointRelation) error {
	return service.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(BucketName))

		data, err := internal.MarshalObject(endpointRelation)
		if err != nil {
			return err
		}

		return bucket.Put(internal.Itob(int(endpointRelation.EndpointID)), data)
	})
}

// UpdateEndpointRelation updates an Endpoint relation object
func (service *Service) UpdateEndpointRelation(EndpointID portainer.EndpointID, endpointRelation *portainer.EndpointRelation) error {
	identifier := internal.Itob(int(EndpointID))
	return internal.UpdateObject(service.db, BucketName, identifier, endpointRelation)
}

// DeleteEndpointRelation deletes an Endpoint relation object
func (service *Service) DeleteEndpointRelation(EndpointID portainer.EndpointID) error {
	identifier := internal.Itob(int(EndpointID))
	return internal.DeleteObject(service.db, BucketName, identifier)
}
