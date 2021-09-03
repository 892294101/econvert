package main

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Tables struct {
	TABLE_SCHEMA *string
	TABLE_NAME   *string
	TABLE_TYPE   *string
	ENGINE       *string
	SIZE         *int
	Method       string
	State        string
	WAIT_TIME    int64
}

type TableSets []*Tables

var (
	Platform  string
	BuildTime string
	GoVersion string
	VERSION   string
)

const (
	NEW_NAME string = "_QNNEW"
	OLD_NAME string = "_QNOLD"
)

var host = flag.String("host", "localhost", "Specify the database host address")
var port = flag.Int("port", 3306, "Specify the database host port")
var user = flag.String("user", "root", "Specify database authentication user")
var password = flag.String("password", "", "Specify database authentication password")
var cdb = flag.String("cdb", "", "Specify the DB to be converted")
var fromengine = flag.String("fromengine", "myisam", "Specify conversion source engine")
var toengine = flag.String("toengine", "innodb", "Specify conversion target engine")
var convert = flag.String("convert", "no", "Perform engine conversion?")
var table = flag.String("table", "", "Specify the tables to be converted. Multiple tables are separated by commas")
var method = flag.String("method", "CTAS", "Select engine conversion method. Available values: CTAS, ALTER")
var size = flag.String("size", "300", "Force alter when the table size is MB?. Only when method is CTAS")
var exclude = flag.String("exclude", "", "Specify excluded tables, multiple tables are separated by commas")
var errcount = flag.String("errcount", "0", "Stop running after reaching the number of times, 0 disabled")
var clean = flag.String("clean", "no", "Clear temporary table")

var session *sql.DB
var alter_size int
var error_count int

func main() {
	PutVersionInfo()
	flag.Parse()
	CheckParmValues()
	runtime.GOMAXPROCS(4)

	dbchar := fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=6s&readTimeout=1200s&writeTimeout=1200s&charset=utf8", *user, *password, *host, *port)
	InitConnection(dbchar)

	var ts TableSets
	if err := ts.LoadTableSet(*table, *exclude); err != nil {
		fmt.Fprintln(os.Stderr, err)
		CloseConnection()
		os.Exit(1)
	}

	ts.PutTableSet()
	if len(ts) > 0 && strings.ToLower(*convert) == "yes" {
		InitTerminal(&ts)
	}
	CloseConnection()

}

/*func (ts *TableSets) IsBreak() bool {
	if len(*ts) > 0 && ts != nil {
		err := termbox.Init()
		if err != nil {
			panic(err)
		}
		defer termbox.Close()
		fmt.Fprintln(os.Stdout, "Press Enter Y to continue, enter ESC or Ctrl+C or N to exit")
		for {
			switch ev := termbox.PollEvent(); ev.Type {
			case termbox.EventKey:
				switch {
				case ev.Key == termbox.KeyEsc || ev.Key == termbox.KeyCtrlC || ev.Key == termbox.KeyCtrlN:
					fmt.Fprintln(os.Stderr, "Exit")
					return false
				case ev.Key == termbox.KeyCtrlY:
					return true
				default:
					fmt.Fprintf(os.Stderr, "Error: %v\n", ev.Key)
				}
			}
		}
	}
	return false
}*/

func ValueAnalyze(par *string, sig chan os.Signal, ts *TableSets) {
	if len(*par) > 0 {
		switch {
		case strings.EqualFold(*par, "y") == true || strings.EqualFold(*par, "yes") == true:
			if err := ts.EngineConvert(); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n\n", err)
				ts.PutTableSet()
				CloseConnection()
				os.Exit(1)
			}
			if len(*ts) > 0 {
				ts.PutTableSet()
				CloseConnection()
				os.Exit(1)
			}
		case strings.EqualFold(*par, "e") == true || strings.EqualFold(*par, "exit") == true:
			CloseConnection()
			fmt.Fprintln(os.Stdout, "")
			os.Exit(1)
		case strings.EqualFold(*par, "h") == true || strings.EqualFold(*par, "help") == true:
			Help()
		case strings.EqualFold(*par, "p") == true:
			ts.PutTableSet()
		case strings.EqualFold(*par, "c") == true:
			PutParam()
			fmt.Fprintf(os.Stdout, "\n")
		case strings.EqualFold(*par, "r") == true:
			if err := ts.ReLoadTableSet(); err != nil {
				fmt.Fprintf(os.Stderr, "Table reload failed: %v\n", err)
				CloseConnection()
				os.Exit(1)
			} else {
				fmt.Fprintf(os.Stdout, "Table reloading is completed. A total of %d tables are loaded\n\n", len(*ts))
			}
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %v\n\n", *par)
		}
	}
	//丢弃Ctrl+C
	if len(sig) > 0 {
		switch <-sig {
		case os.Interrupt:
			fmt.Printf("\n")
		}
	}
}

func InitTerminal(ts *TableSets) {
	reader, sig := InitStd()
	Help()
	for {
		TerminalCli()
		data, _, _ := reader.ReadLine()
		command := strings.TrimSpace(string(data))
		ValueAnalyze(&command, sig, ts)
	}
}

func Help() {
	fmt.Fprintln(os.Stdout, "The available command sets are as follows")
	fmt.Fprintln(os.Stdout, " y    Perform storage engine conversion")
	fmt.Fprintln(os.Stdout, " e    Exit storage engine conversion")
	fmt.Fprintln(os.Stdout, " p    Re output engine conversion table")
	fmt.Fprintln(os.Stdout, " c    Re output database configuration information")
	fmt.Fprintln(os.Stdout, " r    Reload transformation engine table")
	fmt.Fprintf(os.Stdout, " h    Output command information\n\n")
}

func InitStd() (*bufio.Reader, chan os.Signal) {
	reader := bufio.NewReader(os.Stdin)
	signal_receive := make(chan os.Signal, 1)
	signal.Notify(signal_receive, os.Interrupt)
	return reader, signal_receive
}

func TerminalCli() {
	//hn, _ := os.Hostname()
	fmt.Fprintf(os.Stdin, "QNCLI> ")
}

func PutVersionInfo() {
	fmt.Fprintf(os.Stdout, " Build Platform: %s\n", Platform)
	fmt.Fprintf(os.Stdout, "     Build Time: %s\n", BuildTime)
	fmt.Fprintf(os.Stdout, " GoLang Version: %s\n", GoVersion)
	r, err := strconv.ParseFloat(VERSION, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "Program Version: %.2f\n\n", r)

}

func CheckParmValues() {

	if user == nil || len(*user) == 0 {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", "The -user parameter must be specified")
		os.Exit(1)
	}
	if password == nil || len(*password) == 0 {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", "The -password parameter must be specified")
		os.Exit(1)
	}
	if host == nil || len(*host) == 0 {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", "The -host parameter must be specified")
		os.Exit(1)
	}
	if port == nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", "The -port parameter must be specified")
		os.Exit(1)
	}
	if cdb == nil || len(*cdb) == 0 {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", "The -cdb parameter must be specified")
		os.Exit(1)
	}

	if fromengine == nil || len(*fromengine) == 0 {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n\n", "The -fromengine parameter must be specified")
		os.Exit(1)
	} else if strings.ToUpper(*fromengine) != "MYISAM" && strings.ToUpper(*fromengine) != "INNODB" {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n\n", "The -fromengine parameter Only MyISAM and InnoDB are supported")
		os.Exit(1)
	}

	if toengine == nil || len(*toengine) == 0 {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", "The -toengine parameter must be specified")
		os.Exit(1)
	} else if strings.ToUpper(*toengine) != "MYISAM" && strings.ToUpper(*toengine) != "INNODB" {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n\n", "The -toengine parameter Only MyISAM and InnoDB are supported")
		os.Exit(1)
	}

	if *fromengine == *toengine {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", "The source and destination storage engines cannot be the same")
		os.Exit(1)
	}

	if convert != nil && strings.ToLower(*convert) != "no" && strings.ToLower(*convert) != "yes" {
		fmt.Fprintf(os.Stderr, "ERROR: This -convert parameter value is entered illegally: %v\n", *convert)
		os.Exit(1)
	}
	if method != nil && strings.ToUpper(*method) != "CTAS" && strings.ToUpper(*method) != "ALTER" {
		fmt.Fprintf(os.Stderr, "ERROR: This -method parameter value is entered illegally: %v\n", *method)
		os.Exit(1)
	}

	if clean != nil && strings.ToLower(*clean) != "no" && strings.ToLower(*clean) != "yes" {
		fmt.Fprintf(os.Stderr, "ERROR: This -clean parameter value is entered illegally: %v\n", *clean)
		os.Exit(1)
	}
	if size != nil {
		r, err := strconv.Atoi(*size)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: This -size parameter Numerical validation failed, and there may be a bug: %v\n", *size)
			os.Exit(1)
		}
		if r < 0 {
			fmt.Fprintf(os.Stderr, "ERROR: This -size parameter value cannot be less than 0: %v\n", *size)
			os.Exit(1)
		}
		alter_size = r
	}
	if errcount != nil {
		fmt.Println(errcount, *errcount)
		r, err := strconv.Atoi(*errcount)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: This -errcount parameter Numerical validation failed, and there may be a bug: %v\n", *errcount)
			os.Exit(1)
		}
		if r < 0 {
			fmt.Fprintf(os.Stderr, "ERROR: This -errcount parameter value cannot be less than 0: %v\n", *errcount)
			os.Exit(1)
		}
		error_count = r
	}
	PutParam()
}

func PutParam() {
	fmt.Fprintln(os.Stdout, "Database Connection Authentication Information:")
	fmt.Fprintf(os.Stdout, "      HOST: %s\n", *host)
	fmt.Fprintf(os.Stdout, "      Port: %d\n", *port)
	fmt.Fprintf(os.Stdout, "      User: %s\n", *user)
	fmt.Fprintf(os.Stdout, "  PASSWORD: %s\n", *password)
	fmt.Fprintf(os.Stdout, "       CDB: %s\n", *cdb)
	fmt.Fprintf(os.Stdout, "FROMENGINE: %s\n", *fromengine)
	fmt.Fprintf(os.Stdout, "  TOENGINE: %s\n", *toengine)
	fmt.Fprintf(os.Stdout, "   CONVERT: %v\n", *convert)
	fmt.Fprintf(os.Stdout, "     TABLE: %v\n", *table)
	fmt.Fprintf(os.Stdout, "   EXCLUDE: %v\n", *exclude)
	fmt.Fprintf(os.Stdout, "    METHOD: %v\n", *method)
	fmt.Fprintf(os.Stdout, "ERRORCOUNT: %v\n", error_count)
	fmt.Fprintf(os.Stdout, "     CLEAN: %v\n", *clean)
	fmt.Fprintf(os.Stdout, "      SIZE: %v (In CTAS mode, the table is less than %vMB, and the ALTER mode is forced)\n", alter_size, alter_size)
}

func (ts *TableSets) LoadTableSet(tab, exclude string) error {
	var tv string
	if len(tab) > 0 {
		table := strings.Split(tab, ",")
		for i, s := range table {
			if i == 0 {
				tv += fmt.Sprintf("'%s'", s)
			} else {
				tv += fmt.Sprintf(",'%s'", s)
			}
		}
	}

	var ev string
	if len(exclude) > 0 {
		etab := strings.Split(exclude, ",")
		for i, s := range etab {
			if i == 0 {
				ev += fmt.Sprintf("'%s'", s)
			} else {
				ev += fmt.Sprintf(",'%s'", s)
			}
		}
	}

	ltbsql := `select TABLE_SCHEMA,
       TABLE_NAME,
       TABLE_TYPE,
       ENGINE,
       round(DATA_LENGTH / 1024 / 1024) SIZE,
       'Ready' STATE
  from information_schema.tables
 where TABLE_TYPE = 'BASE TABLE'
   and TABLE_SCHEMA = ?
   and upper(ENGINE) = upper(?)`
	ltbsql += fmt.Sprintf("\n   and instr(TABLE_NAME,'%s')=0 \n", OLD_NAME)
	if len(tv) > 0 {
		ltbsql += fmt.Sprintf("   and TABLE_NAME in (%s) \n", tv)
	}
	if len(ev) > 0 {
		ltbsql += fmt.Sprintf("   and TABLE_NAME NOT IN (%s) \n", ev)
	}
	ltbsql += ` order by size desc`

	//fmt.Fprintf(os.Stdout, "SQL> %s;\n", ltbsql)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	res, err := session.QueryContext(ctx, ltbsql, *cdb, *fromengine)
	defer res.Close()
	if err != nil {
		return errors.New(fmt.Sprintf("ERROR: Query MySQL list: %s", err.Error()))
	}

	for res.Next() {
		var t Tables
		if err := res.Scan(&t.TABLE_SCHEMA, &t.TABLE_NAME, &t.TABLE_TYPE, &t.ENGINE, &t.SIZE, &t.State); err != nil {
			return errors.New(fmt.Sprintf("ERROR: Data retrieval failed: %s", err.Error()))
		}
		*ts = append(*ts, &t)
	}
	return nil
}

func (ts *TableSets) ReLoadTableSet() error {
	*ts = nil
	return ts.LoadTableSet(*table, *exclude)
}

func (ts *TableSets) EngineConvert() error {
	var error int
	for _, set := range *ts {
		ddl, err := QueryTableStructure(*set.TABLE_SCHEMA, *set.TABLE_NAME)
		if err != nil {
			return errors.New(fmt.Sprintf("Table structure query failed: %s %s\n\n", *set.TABLE_SCHEMA, *set.TABLE_NAME))
		} else {
			switch {
			case strings.ToUpper(*method) == "ALTER":
				set.WAIT_TIME, err = ConvertEngineAlter(*ddl, *set.TABLE_SCHEMA, *set.TABLE_NAME, *fromengine, *toengine)
				set.Method = "ALTER"
				if err != nil {
					set.State = "FAILED"
					if error_count > 0 {
						error += 1
						fmt.Fprintf(os.Stderr, "%s\n\n", err)
					} else {
						return err
					}
				} else {
					set.State = "SUCCESS"
				}
				break
			case strings.ToUpper(*method) == "CTAS":
				if *set.SIZE <= alter_size {
					set.WAIT_TIME, err = ConvertEngineAlter(*ddl, *set.TABLE_SCHEMA, *set.TABLE_NAME, *fromengine, *toengine)
					set.Method = "ALTER"
					if err != nil {
						set.State = "FAILED"
						if error_count > 0 {
							error += 1
							fmt.Fprintf(os.Stderr, "%s\n\n", err)
						} else {
							return err
						}
					} else {
						set.State = "SUCCESS"
					}
				} else {
					set.WAIT_TIME, err = ConvertEngineCTAS(*ddl, *set.TABLE_SCHEMA, *set.TABLE_NAME, *fromengine, *toengine)
					set.Method = "CTAS"
					if err != nil {
						set.State = "FAILED"
						if error_count > 0 {
							error += 1
							fmt.Fprintf(os.Stderr, "%s\n\n", err)
						} else {
							return err
						}
					} else {
						set.State = "SUCCESS"
					}
				}
				break
			}
			if error_count > 0 {
				if error == error_count {
					return errors.New(fmt.Sprintf("The error count reached %d with a maximum of %d", error, error_count))
				}
			}

		}
	}
	return nil
}

func (ts *TableSets) PutTableSet() {
	if len(*ts) == 0 {
		fmt.Fprintf(os.Stdout, "WARN: The %s database did not query any tables of the %s storage engine\n\n", *cdb, *fromengine)
	} else {
		fmt.Fprintln(os.Stdout, "Database table information statistics:")

		var Seq int = 3
		var LENGTH_TABLE_SCHEMA = 12
		var LENGTH_TABLE_NAME = 10
		var LENGTH_TABLE_TYPE = 10
		var LENGTH_ENGINE = 6
		var LENGTH_SIZE = 7
		var LENGTH_METHOD = 6
		var LENGTH_STATE = 8
		var LENGTH_WAIT_TIME = 10

		for i, set := range *ts {
			switch {
			case len(string(i)) > Seq:
				Seq = len(string(i))
			case len(*set.TABLE_SCHEMA) > LENGTH_TABLE_SCHEMA:
				LENGTH_TABLE_SCHEMA = len(*set.TABLE_SCHEMA)
			case len(*set.TABLE_NAME) > LENGTH_TABLE_NAME:
				LENGTH_TABLE_NAME = len(*set.TABLE_NAME)
			case len(*set.TABLE_TYPE) > LENGTH_TABLE_TYPE:
				LENGTH_TABLE_TYPE = len(*set.TABLE_TYPE)
			case len(*set.ENGINE) > LENGTH_ENGINE:
				LENGTH_ENGINE = len(*set.ENGINE)
			case len(string(*set.SIZE)) > LENGTH_SIZE:
				LENGTH_SIZE = len(string(*set.SIZE))
			case len(set.Method) > LENGTH_METHOD:
				LENGTH_METHOD = len(set.Method)
			case len(set.State) > LENGTH_STATE:
				LENGTH_STATE = len(set.State)
			case len(string(set.WAIT_TIME)) > LENGTH_WAIT_TIME:
				LENGTH_WAIT_TIME = len(string(set.WAIT_TIME))
			}
		}

		var count int = len(*ts)
		for i, set := range *ts {
			fm := fmt.Sprintf("| %%%dv | %%%dv | %%%dv | %%%dv | %%%dv | %%%dv | %%%dv | %%%dv | %%%dv |\n", Seq, LENGTH_TABLE_SCHEMA, LENGTH_TABLE_NAME, LENGTH_TABLE_TYPE, LENGTH_ENGINE, LENGTH_SIZE, LENGTH_METHOD, LENGTH_STATE, LENGTH_WAIT_TIME)
			head := fmt.Sprintf("+%%%dv+%%%dv+%%%dv+%%%dv+%%%dv+%%%dv+%%%dv+%%%dv+%%%dv+\n", Seq, LENGTH_TABLE_SCHEMA, LENGTH_TABLE_NAME, LENGTH_TABLE_TYPE, LENGTH_ENGINE, LENGTH_SIZE, LENGTH_METHOD, LENGTH_STATE, LENGTH_WAIT_TIME)
			if i == 0 {
				fmt.Fprintf(os.Stdout, head, StrCompletion(Seq, "-"), StrCompletion(LENGTH_TABLE_SCHEMA, "-"), StrCompletion(LENGTH_TABLE_NAME, "-"), StrCompletion(LENGTH_TABLE_TYPE, "-"), StrCompletion(LENGTH_ENGINE, "-"), StrCompletion(LENGTH_SIZE, "-"), StrCompletion(LENGTH_METHOD, "-"), StrCompletion(LENGTH_STATE, "-"), StrCompletion(LENGTH_WAIT_TIME, "-"))
				fmt.Fprintf(os.Stdout, fm, "SEQ", "TABLE_SCHEMA", "TABLE_NAME", "TABLE_TYPE", "ENGINE", "SIZE_MB", "METHOD", "STATE", "WAIT_TIME")
				fmt.Fprintf(os.Stdout, head, StrCompletion(Seq, "-"), StrCompletion(LENGTH_TABLE_SCHEMA, "-"), StrCompletion(LENGTH_TABLE_NAME, "-"), StrCompletion(LENGTH_TABLE_TYPE, "-"), StrCompletion(LENGTH_ENGINE, "-"), StrCompletion(LENGTH_SIZE, "-"), StrCompletion(LENGTH_METHOD, "-"), StrCompletion(LENGTH_STATE, "-"), StrCompletion(LENGTH_WAIT_TIME, "-"))
			}
			fmt.Fprintf(os.Stdout, fm, i+1, *set.TABLE_SCHEMA, *set.TABLE_NAME, *set.TABLE_TYPE, *set.ENGINE, *set.SIZE, set.Method, set.State, set.WAIT_TIME)
			if count == i+1 {
				fmt.Fprintf(os.Stdout, head+"\n", StrCompletion(Seq, "-"), StrCompletion(LENGTH_TABLE_SCHEMA, "-"), StrCompletion(LENGTH_TABLE_NAME, "-"), StrCompletion(LENGTH_TABLE_TYPE, "-"), StrCompletion(LENGTH_ENGINE, "-"), StrCompletion(LENGTH_SIZE, "-"), StrCompletion(LENGTH_METHOD, "-"), StrCompletion(LENGTH_STATE, "-"), StrCompletion(LENGTH_WAIT_TIME, "-"))
			}
		}
	}

}

func QueryTableStructure(schema, table string) (*string, error) {
	ltbsql := fmt.Sprintf("show create table `%s`.`%s`", schema, table)
	/*ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()*/
	res, err := session.Query(ltbsql)
	defer res.Close()
	if err != nil {
		return nil, err
	}
	var tab string
	var ddl string
	for res.Next() {
		if err := res.Scan(&tab, &ddl); err != nil {
			return nil, err
		}
	}
	return &ddl, nil
}

func InitConnection(dbchar string) {
	db, err := sql.Open("mysql", dbchar)
	fmt.Printf("       DSN: %s\n", dbchar)
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err.Error())
		os.Exit(1)
	} else {
		fmt.Fprintln(os.Stdout, "MySQL Driver registration completed")
	}
	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "MySQL connection failed: %s\n", err.Error())
		os.Exit(1)
	} else {
		fmt.Fprintln(os.Stdout, "MySQL connection succeeded")
		fmt.Fprintln(os.Stdout, "")
	}
	session = db
}

func CloseConnection() {
	if session != nil {
		if err := session.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Mysql database connection closing failed: %s", err.Error())
			os.Exit(1)
		}
	} else {
		fmt.Fprintln(os.Stderr, "Null pointer encountered while closing connection")
		os.Exit(1)
	}
}

func ConvertEngineCTAS(ddl, schema, table, fromengine, toengine string) (int64, error) {
	var total int64

	source_table := fmt.Sprintf("`%s`.`%s`", schema, table)
	target_table := fmt.Sprintf("`%s`.`%s%s`", schema, table, NEW_NAME)
	tmp := strings.ReplaceAll(ddl, fmt.Sprintf("`%s`", table), target_table)
	var readsql string
	switch strings.ToUpper(fromengine) {
	case "MYISAM":
		readsql = strings.ReplaceAll(tmp, "ENGINE=MyISAM", fmt.Sprintf("ENGINE=%s", toengine))
	case "INNODB":
		readsql = strings.ReplaceAll(tmp, "ENGINE=InnoDB", fmt.Sprintf("ENGINE=%s", toengine))
	default:
		return total, errors.New(fmt.Sprintf("This %s storage engine does not support transformations", fromengine))
	}

	fmt.Fprintf(os.Stdout, "SQL> %s;\n", readsql)
	begin := time.Now()
	_, err := session.ExecContext(context.TODO(), readsql)
	tut := GetTimeDiffer(begin, time.Now())
	if err != nil {
		return 0, errors.New(fmt.Sprintf("Create table failed: %s. %s", target_table, err))
	} else {
		total += tut
		fmt.Fprintf(os.Stdout, "Table Created (%d Millisecond)\n\n", tut)
	}

	insql := fmt.Sprintf("INSERT INTO %s SELECT * FROM %s", target_table, source_table)
	fmt.Fprintf(os.Stdout, "SQL> %s;\n", insql)
	begin = time.Now()
	res, err := session.ExecContext(context.TODO(), insql)
	tut = GetTimeDiffer(begin, time.Now())
	if err != nil {
		return total, errors.New(fmt.Sprintf("Failed to insert data: (%s--->%s). ERROR: %s", source_table, target_table, err))
	} else {
		if ct, err := res.RowsAffected(); err != nil {
			return total, errors.New(fmt.Sprintf("Data row Affected Get failed. ERROR: %s", err))
		} else {
			total += tut
			fmt.Fprintf(os.Stdout, "Insert Data OK, %d rows affected (%d Millisecond)\n\n", ct, tut)
		}
	}

	oldtab := fmt.Sprintf("`%s`.`%s%s`", schema, table, OLD_NAME)
	oldsql := fmt.Sprintf("ALTER TABLE %s RENAME TO %s", source_table, oldtab)
	fmt.Fprintf(os.Stdout, "SQL> %s;\n", oldsql)
	/*ctx3, cancel3 := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel3()*/
	begin = time.Now()
	_, err = session.Exec(oldsql)
	tut = GetTimeDiffer(begin, time.Now())
	if err != nil {
		return total, err
	} else {
		total += tut
		fmt.Fprintf(os.Stdout, "Table altered (%d Millisecond)\n\n", tut)
	}

	newsql := fmt.Sprintf("ALTER TABLE %s RENAME TO %s", target_table, source_table)
	fmt.Fprintf(os.Stdout, "SQL> %s;\n", newsql)
	begin = time.Now()
	_, err = session.Exec(newsql)
	tut = GetTimeDiffer(begin, time.Now())
	if err != nil {
		return total, err
	} else {
		total += tut
		fmt.Fprintf(os.Stdout, "Table altered (%d Millisecond)\n\n", tut)
	}

	if strings.EqualFold(*clean, "YES") == true {
		dropsql := fmt.Sprintf("DROP TABLE %s", oldtab)
		fmt.Fprintf(os.Stdout, "SQL> %s;\n", dropsql)
		begin = time.Now()
		_, err = session.Exec(dropsql)
		tut = GetTimeDiffer(begin, time.Now())
		if err != nil {
			return total, err
		} else {
			total += tut
			fmt.Fprintf(os.Stdout, "Table Drop (%d Millisecond)\n\n", tut)
		}
	}
	return total, nil
}

func ConvertEngineAlter(ddl, schema, table, fromengine, toengine string) (int64, error) {
	execsql := fmt.Sprintf("alter table `%s`.`%s` engine=%s", schema, table, toengine)
	fmt.Fprintf(os.Stdout, "SQL> %s;\n", execsql)
	begin := time.Now()
	res, err := session.Exec(execsql)
	tut := GetTimeDiffer(begin, time.Now())
	if err != nil {
		return tut, err
	} else {
		if ct, err := res.RowsAffected(); err != nil {
			return tut, errors.New(fmt.Sprintf("Table altered. Data row Affected Get failed. ERROR: %s", err))
		} else {
			fmt.Fprintf(os.Stdout, "Table altered. %d rows affected (%d Millisecond)\n\n", ct, tut)
		}
	}
	return tut, nil
}

func GetTimeDiffer(begin, current time.Time) int64 {

	b := begin.Local().Format("2006-01-02 15:04:05.000000")
	c := current.Local().Format("2006-01-02 15:04:05.000000")
	t1, err1 := time.ParseInLocation("2006-01-02 15:04:05.000000", b, time.Local)
	t2, err2 := time.ParseInLocation("2006-01-02 15:04:05.000000", c, time.Local)
	if err1 == nil && err2 == nil && t1.Before(t2) {
		/*diff := t2.Unix() - t1.Unix() //
		return diff / 3600 * 60 * 60 * 1000*/
		return t2.Sub(t1).Milliseconds()
	}
	return 0
}

func StrCompletion(length int, str string) string {
	var r string
	for i := 0; i < length+2; i++ {
		r = str + r
	}
	return r
}
