package rating

type Service interface {
	Submit(rating *Rating) error
	DoctorRatings(doctorID uint) ([]Rating, error)
	MyRatings(userID uint) ([]Rating, error)
	HasRatedAppointment(userID, appointmentID uint) (bool, error)
	HideRating(ratingID uint, hiddenBy uint, reason string) error
	RestoreRating(ratingID uint) error
	GetAllForAdmin(doctorID uint, minScore, maxScore int, page, limit int) ([]Rating, int64, error)
	GetByID(ratingID uint) (*Rating, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Submit(r *Rating) error {
	if err := s.repo.CreateRating(r); err != nil {
		return err
	}
	// Recalculate doctor's average rating in the background
	go s.repo.UpdateDoctorAverageRating(r.DoctorID)
	return nil
}

func (s *service) DoctorRatings(doctorID uint) ([]Rating, error) {
	return s.repo.GetByDoctorID(doctorID)
}

func (s *service) MyRatings(userID uint) ([]Rating, error) {
	return s.repo.GetByUserID(userID)
}

func (s *service) HasRatedAppointment(userID, appointmentID uint) (bool, error) {
	return s.repo.HasRatedAppointment(userID, appointmentID)
}

func (s *service) HideRating(ratingID uint, hiddenBy uint, reason string) error {
	r, err := s.repo.GetByID(ratingID)
	if err != nil {
		return err
	}
	if err := s.repo.HideRating(ratingID, hiddenBy, reason); err != nil {
		return err
	}
	go s.repo.UpdateDoctorAverageRating(r.DoctorID)
	return nil
}

func (s *service) RestoreRating(ratingID uint) error {
	r, err := s.repo.GetByID(ratingID)
	if err != nil {
		return err
	}
	if err := s.repo.RestoreRating(ratingID); err != nil {
		return err
	}
	go s.repo.UpdateDoctorAverageRating(r.DoctorID)
	return nil
}

func (s *service) GetAllForAdmin(doctorID uint, minScore, maxScore int, page, limit int) ([]Rating, int64, error) {
	return s.repo.GetAllForAdmin(doctorID, minScore, maxScore, page, limit)
}

func (s *service) GetByID(ratingID uint) (*Rating, error) {
	return s.repo.GetByID(ratingID)
}
