package services

import (
	"time"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/admin"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/authorization"
)

type AdminService struct {
	repository admin.AdminRepository

	// Services used when creating an admin.
	authService *AuthorizationService
}

// Create a new AdminService.
func NewAdminService(repository admin.AdminRepository, authService *AuthorizationService) *AdminService {
	return &AdminService{
		repository:  repository,
		authService: authService,
	}
}

func (s *AdminService) Create(name string, expiry time.Time) (admin.Admin, error) {
	a := admin.MakeAdmin(name, expiry)
	a, err := s.repository.Create(a)
	if err != nil {
		return admin.Admin{}, err
	}

	a.AuthorizationToken, err = s.authService.CreateToken(authorization.AdminToken, a.ID, a.Expiry)
	if err != nil {
		return admin.Admin{}, err
	}

	return a, nil
}

func (s *AdminService) Find(admin admin.Admin) (admin.Admin, error) {
	return s.repository.Find(admin)
}

func (s *AdminService) GetAll() ([]admin.Admin, error) {
	return s.repository.GetAll()
}

func (s *AdminService) Delete(admin admin.Admin) error {
	return s.repository.Delete(admin)
}

func (s *AdminService) Reactivate(a admin.Admin) (admin.Admin, error) {
	a, err := s.repository.Find(a)
	if err != nil {
		return admin.Admin{}, err
	}

	a.Reactivate()

	a, err = s.repository.Update(a)
	if err != nil {
		return admin.Admin{}, err
	}

	a.AuthorizationToken, err = s.authService.CreateToken(authorization.AdminToken, a.ID, a.Expiry)
	if err != nil {
		return admin.Admin{}, err
	}

	return a, nil
}

func (s *AdminService) SetExpiry(a admin.Admin, expiry time.Time) (admin.Admin, error) {
	a, err := s.repository.Find(a)
	if err != nil {
		return admin.Admin{}, err
	}

	a.SetExpiry(expiry)
	return s.repository.Update(a)
}
