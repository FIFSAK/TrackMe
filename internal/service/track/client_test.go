package track

//
//import (
//	"TrackMe/internal/domain/client"
//	"TrackMe/internal/domain/contract"
//	"TrackMe/pkg/store"
//	"context"
//	"errors"
//	"fmt"
//	"github.com/stretchr/testify/mock"
//	"github.com/stretchr/testify/suite"
//	"github.com/testcontainers/testcontainers-go"
//	"go.mongodb.org/mongo-driver/mongo"
//	"go.mongodb.org/mongo-driver/mongo/options"
//	"log"
//	"testing"
//	"time"
//)
//
//type TestDatabase struct {
//	DbInstance *mongo.Database
//	DbAddress  string
//	container  testcontainers.Container
//}
//
//func SetupTestDatabaseWithName(name string) *TestDatabase {
//	ctx, _ := context.WithTimeout(context.Background(), time.Second*60)
//	container, dbInstance, dbAddr, err := createMongoContainer(ctx, name)
//	if err != nil {
//		log.Fatal("failed to setup test", err)
//	}
//
//	return &TestDatabase{
//		container:  container,
//		DbInstance: dbInstance,
//		DbAddress:  dbAddr,
//	}
//}
//
//func (tdb *TestDatabase) TearDown() {
//	_ = tdb.container.Terminate(context.Background())
//}
//
//func NewMongoDatabase(uri string, database string) (*mongo.Database, error) {
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
//	if err != nil {
//		return nil, err
//	}
//
//	db := client.Database(database)
//
//	return db, nil
//}
//
//func createMongoContainer(ctx context.Context, name string) (testcontainers.Container, *mongo.Database, string, error) {
//	var env = map[string]string{
//		"MONGO_INITDB_ROOT_USERNAME": "root",
//		"MONGO_INITDB_ROOT_PASSWORD": "pass",
//		"MONGO_INITDB_DATABASE":      name,
//	}
//	var port = "27017/tcp"
//
//	req := testcontainers.GenericContainerRequest{
//		ContainerRequest: testcontainers.ContainerRequest{
//			Image:        "mongo",
//			ExposedPorts: []string{port},
//			Env:          env,
//		},
//		Started: true,
//	}
//	container, err := testcontainers.GenericContainer(ctx, req)
//	if err != nil {
//		return container, nil, "", fmt.Errorf("failed to start container: %v", err)
//	}
//
//	p, err := container.MappedPort(ctx, "27017")
//	if err != nil {
//		return container, nil, "", fmt.Errorf("failed to get container external port: %v", err)
//	}
//
//	log.Println("mongo container ready and running at port: ", p.Port())
//
//	uri := fmt.Sprintf("mongodb://root:pass@localhost:%s", p.Port())
//	db, err := NewMongoDatabase(uri, name)
//	if err != nil {
//		return container, db, uri, fmt.Errorf("failed to establish database connection: %v", err)
//	}
//
//	return container, db, uri, nil
//}
//
//type RepositorySuite struct {
//	suite.Suite
//	repository   client.Repository
//	testDatabase *TestDatabase
//}
//
//type SetupAllSuite interface {
//	SetupSuite()
//}
//
//type TearDownAllSuite interface {
//	TearDownSuite()
//}
//
//func (suite *RepositorySuite) SetupSuite() {
//	suite.testDatabase = SetupTestDatabaseWithName("cliets_test_db")
//	suite.repository = suite.repository.NewClientRepository(suite.testDatabase.DbInstance)
//}
//
//func (suite *RepositorySuite) TearDownSuite() {
//	suite.testDatabase.container.Terminate(context.Background())
//}
