package main

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"net/url"
	"os"
	"strings"
)

func envOrDefault(envName string, def string) string {
	val := os.Getenv(envName)
	if val == "" {
		return def
	}
	return val
}

func main(){
	logger := logrus.New()
	if envOrDefault("CHASQUID_ALIAS_DEBUG", "0") == "1" {
		logger.SetLevel(logrus.DebugLevel)
	}
	if len(os.Args) != 2 {
		logger.Errorf("invalid usage: please provide an email address as a parameter")
		os.Exit(2)
	}

	if os.Args[1] == "-h" {
		fmt.Printf("Usage: %s EMAIL-ADDRESS\n", os.Args[0])
		os.Exit(2)
	}

	emailAddr := os.Args[1]
	emailAddrSplit := strings.SplitAfter(emailAddr, "@")

	if len(emailAddrSplit) != 2 {
		logger.Errorf("invalid email address provided!")
		os.Exit(2)
	}

	localPart, domain := strings.Replace(emailAddrSplit[0], "@", "", -1), emailAddrSplit[1]

	mysqlUrl := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		url.QueryEscape(envOrDefault("DOVECOT_DB_USER", "root")),
		url.QueryEscape(envOrDefault("DOVECOT_DB_PASSWORD", "")),
		url.QueryEscape(envOrDefault("DOVECOT_DB_HOST", "localhost")),
		url.QueryEscape(envOrDefault("DOVECOT_DB_PORT", "3306")),
		url.QueryEscape(envOrDefault("DOVECOT_DB_NAME", "dovecot")),
	)

	db, err := sqlx.Open("mysql", mysqlUrl)
	defer db.Close()

	if err != nil {
		logger.Errorf("unable to open connection to db: %v", err)
		os.Exit(3)
	}

	rows, err := db.Query(`SELECT
		GROUP_CONCAT(DISTINCT alias_recipients.recipient_address SEPARATOR ', ') as recipients
		FROM
			aliases
		INNER JOIN alias_recipients ON
			aliases.id = alias_recipients.alias_id
		INNER JOIN domains ON
			aliases.domain_id = domains.id
		WHERE
			aliases.local_part = ?
			AND domains.domain  = ?
			AND aliases.active = 1
			AND domains.active = 1;`,
		localPart,
		domain,
	)
	if err != nil {
		logger.Errorf("unable to run query: %v", err)
		os.Exit(4)
	}

	var recipients *string
	if !rows.Next() {
		logger.Debugf("alias %s not found", emailAddr)
		os.Exit(1)
	}
	err = rows.Scan(&recipients)
	if err != nil {
		logger.Errorf("unable to get recipients: %v", err)
		os.Exit(5)
	}

	if recipients == nil {
		logger.Debugf("alias %s not found", emailAddr)
		os.Exit(1)
	}

	fmt.Printf("%s", *recipients)
	os.Exit(0)
}
