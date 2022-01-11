package gott

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

const (
	dateFormat          = "2006-01-02"
	dateFormatShort     = "01-02"
	datetimeFormat      = "2006-01-02 15:04:05"
	datetimeFormatShort = "01-02 15:04"
	timeFormat          = "15:04"
)

var database Database

func init() {

	viper.SetConfigName(".gottrc")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME")

	viper.SetDefault(ConfDatabaseName, "db.json")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("[WARNING] ", err.Error())
	}

	databaseName := viper.GetString(ConfDatabaseName)
	database = NewDatabaseJson(databaseName)
	database.Load()
}

func Execute() {
	defer database.Save()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
