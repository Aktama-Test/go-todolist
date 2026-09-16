package main

import (
 "context"
 "log"
 "dep6542/dns"
)

func main() {
 if err := dns.SetupResolved(context.Background()); err != nil { log.Fatal(err) }
}
