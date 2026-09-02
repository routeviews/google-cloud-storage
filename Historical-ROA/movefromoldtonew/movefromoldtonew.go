package movefromoldtonew

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gidoBOSSftw5731/log"
	"github.com/jackc/pgx/v5"
)

//this should be done in SQL but I am so done with that

// the only point of this
func Main(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	connStr := fmt.Sprintf("postgres://postgres:%s@%s/roas", dbpass, dbip)
	db, err := pgx.Connect(ctx, connStr)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close(ctx)

	rows, err := db.Query(ctx, "SELECT DISTINCT inserttime FROM roas")
	if err != nil {
		log.Fatalln(err)
	}
	defer rows.Close()

	var times []time.Time
	for rows.Next() {
		var t time.Time
		rows.Scan(&t)
		times = append(times, t)
	}

	txn, err := db.Begin(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	defer txn.Rollback(ctx)

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

		r, err := db.Query(ctx, "SELECT DISTINCT asn, prefix, mask, ta, maxlen FROM roas WHERE inserttime = $1", t)
		if err != nil {
			log.Fatalln(err)
		}

		for r.Next() {
			var b temp
			r.Scan(&b.asn, &b.prefix, &b.mask, &b.ta, &b.maxlen)

			buf = append(buf, b)
		}
		r.Close()
		debug++
	}

	debug = 0
	for _, r := range buf {
		debug++
		println(debug)

		ra, err := txn.Exec(ctx, `UPDATE roas_arr
		SET inserttimes = unnest(array_append(inserttimes, $1))
		WHERE asn = $2 AND prefix = $3 AND maxlen = $4 AND ta = $5 AND mask = $6`,
			r.t, r.asn, r.prefix, r.maxlen, r.ta, r.mask)
		if err != nil {
			log.Fatalln(err)
		}

		switch ra.RowsAffected() {
		case 0:
			_, err = txn.Exec(ctx, `INSERT INTO roas_arr(asn, prefix, maxlen, ta, mask, inserttimes)
			VALUES ($1, $2, $3, $4, $5, $6)`,
				r.asn, r.prefix, r.maxlen,
				r.ta, r.mask, []time.Time{r.t})
			if err != nil {
				log.Fatalln(err)
			}
		}
		println("foo3")
	}

	err = txn.Commit(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Success!")

}
