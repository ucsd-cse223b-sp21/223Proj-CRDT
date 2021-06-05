// Sample main accepting flag inputs for executable

package document

import (
	"flag"
	"log"
)

var (
//addr    = flag.String("addr", "", "address of server")
//verbose = flag.Bool("v", false, "verbose logging")
)

func noError(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func main() {
	flag.Parse()
}
