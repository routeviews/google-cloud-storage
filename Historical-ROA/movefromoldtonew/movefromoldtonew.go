package movefromoldtonew

import (
	"net/http"
	"os"
	"time"

	"github.com/gidoBOSSftw5731/log"
	"github.com/jackc/pgx"
)

//this should be done in SQL but I am so done with that

// the only point of this
func Main(w http.ResponseWriter, r *http.Request) {

	log.SetCallDepth(4)

	http.Error(w, "can't hijack rw", 200)
	hj, _ := w.(http.Hijacker)
	conn, _, _ := hj.Hijack()
	conn.Close()

	// open SQL connection
	var err error
	dbpass := os.Getenv("DB_PASSWORD")
	if dbpass == "" {
		dbpass = "datboifff"
	}

	dbip := os.Getenv("DB_ADDR")
	if dbip == "" {
		dbip = "/cloudsql/historical-roas:us-east1:history3"
	}

	db, err := pgx.Connect(pgx.ConnConfig{
		Host:     dbip,
		User:     "postgres",
		Password: dbpass,
		Database: "roas",
	})
	if err != nil {
		log.Fatalln(err)
	}

	rows, err := db.Query("SELECT DISTINCT inserttime FROM roas")
	if err != nil {
		log.Fatalln(err)
	}

	var times []time.Time
	for rows.Next() {
		var t time.Time
		rows.Scan(&t)
		times = append(times, t)
	}

	txn, err := db.Begin()
	if err != nil {
		log.Fatalln(err)
	}

	type temp struct {
		asn, prefix, ta string
		mask, maxlen    int
		t               time.Time
		update          bool
	}
	var debug int
	var buf []temp
	for _, t := range times {
		println(debug)

		r, err := db.Query("SELECT DISTINCT asn, prefix, mask, ta, maxlen FROM roas WHERE inserttime = $1", t)
		if err != nil {
			log.Fatalln(err)
		}

		for r.Next() {
			var b temp
			r.Scan(&b.asn, &b.prefix, &b.mask, &b.ta, &b.maxlen)

			buf = append(buf, b)
		}
		debug++
	}

	/*	debug = 0
		for _, b := range buf {
			debug++
			println(debug)
			err = db.QueryRow("SELECT FROM roas WHERE asn = $1 AND prefix = $2 AND mask = $3 AND ta = $4 AND maxlen = $5",
				b.asn, b.prefix, b.mask, b.ta, b.maxlen).Scan()
			println("foo2")
			switch err {
			case pgx.ErrNoRows:
				b.update = true
			case nil:
				b.update = false
			default:
				log.Fatalln("query failed", err)
			}
			buf2 = append(buf2, b)
		}*/

	debug = 0
	for _, r := range buf {
		debug++
		println(debug)

		ra, err := txn.Exec(`UPDATE roas_arr
		SET inserttimes = unnest(array_append(inserttimes, $1))
		WHERE asn = $2 AND prefix = $3 AND maxlen = $4 AND ta = $5 AND mask = $6`,
			r.t, r.asn, r.prefix, r.maxlen, r.ta, r.mask)
		if err != nil {
			log.Fatalln(err)
		}

		switch ra.RowsAffected() {
		case 0:
			_, err = txn.Exec(`INSERT INTO roas_arr(asn, prefix, maxlen, ta, mask, inserttimes)
			VALUES ($1, $2, $3, $4, $5, $6)`,
				r.asn, r.prefix, r.maxlen,
				r.ta, r.mask, []time.Time{r.t})
			if err != nil {
				log.Fatalln(err)
			}
		}
		println("foo3")
	}

	err = txn.Commit()
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Success!")

}
