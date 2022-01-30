package user

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
	uuid "github.com/nu7hatch/gouuid"
)

type User struct {
	Age      int      `json:"age"`
	Email    string   `json:"email"`
	Id       string   `json:"id"`
	Name     string   `json:"name"`
	NickName string   `json:"nick_name"`
	Matches  []string `json:"matches"`
}

type Service struct {
	table dynamo.Table
}

func NewService(tableName, region string) *Service {
	sess, _ := session.NewSession()
	db := dynamo.New(sess, &aws.Config{Region: aws.String(region)})
	table := db.Table(tableName)

	return &Service{table: table}
}

func (s Service) Create(user User) (User, error) {
	users, err := s.GetByEmail(user.Email)
	if err != nil {
		return User{}, err
	}
	if len(users) > 0 {
		return user, errors.New("user with this email is already created")
	}

	user.Id = CreateId()
	return user, s.table.Put(user).Run()
}

func (s Service) AddMatch(user User, matchId string) (User, error) {
	user.Matches = append(user.Matches, matchId)
	return user, s.table.Update("id", user.Id).Set("matches", user.Matches).Value(&user)
}

func (s Service) GetAll() (users []User, err error) {
	err = s.table.Scan().All(&users)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (s Service) GetByEmail(email string) (user []User, err error) {
	err = s.table.Get("email", email).Index("email-index").All(&user)
	if err != nil {
		return user, err
	}
	return user, nil
}

func CreateId() (id string) {
	v4, _ := uuid.NewV4()
	return v4.String()
}
