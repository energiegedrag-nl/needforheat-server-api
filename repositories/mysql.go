package repositories

import (
	"context"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// Create a new database connection that can be used by repositories.
func NewDatabaseConnection(dsn string) (*gorm.DB, error) {
	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		return nil, err
	}

	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.Params = map[string]string{"charset": "utf8mb4"}

	dsn = cfg.FormatDSN()
	return gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
}

// Create a new database connection and perform a migration.
// Try until connection succeeds or context is done.
func NewDatabaseConnectionAndMigrate(ctx context.Context, dsn string) (db *gorm.DB, err error) {
	_, ok := ctx.Deadline()
	if !ok {
		logrus.Warn("no deadline was set for making database connection. we will try indefinately")
	}

	logrus.Info("connecting to database")

	for {
		db, err = NewDatabaseConnection(dsn)
		if err == nil {
			err = db.AutoMigrate(
				&AppModel{},
				&CloudFeedTypeModel{},
				&DataSourceListModel{},
				&CampaignModel{},
				&DataSourceTypeModel{},
				&AccountModel{},
				&CloudFeedModel{},
				&PropertyModel{},
				&UploadModel{},
				&DeviceTypeModel{},
				&DeviceModel{},
				&MeasurementModel{},
				&DataSourceListItems{},
				&EnergyQueryTypeModel{},
				&EnergyQueryModel{},
				&APIKeyModel{},
				&InventoryDeviceModel{},
				&InventoryEventModel{},
				&VaultBlobModel{},
				&CampaignMailModel{},
				&PropertyLabelModel{},
			)
			if err != nil {
				return db, err
			}
			return db, repairEnergyQueryUploadTimes(db)
		}

		select {
		case <-time.After(time.Second): // Wait for 1 second before we loop again.
		case <-ctx.Done():
			return // Return with the database and error.
		}
	}
}

// Uploads of energy queries were stored without a server time (before MakeEnergyQuery set it):
// give them the activation time of their energy query, the moment they were received. Runs on
// every start; it changes nothing once they are repaired, and a failure only gives a warning.
func repairEnergyQueryUploadTimes(db *gorm.DB) error {
	result := db.Exec(`UPDATE upload u JOIN energy_query eq ON u.instance_id = eq.id
		SET u.server_time = eq.activated_at
		WHERE u.instance_type = 'energy_query' AND eq.activated_at IS NOT NULL
		AND (u.server_time IS NULL OR u.server_time < '1971-01-01')`)
	if result.Error != nil {
		logrus.WithError(result.Error).Warn("could not set the server time of energy-query uploads")
		return nil
	}
	if result.RowsAffected > 0 {
		logrus.Infof("set the server time of %d energy-query uploads", result.RowsAffected)
	}
	return nil
}
