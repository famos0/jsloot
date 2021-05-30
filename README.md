# JSLoot

Looting URLs, IPv4 addresses, base64 encoded stuff and aws-keys from JavaScript

## Install

```bash
go get -u github.com/famos0/jsloot
```

## Use

```bash
$ jsloot -h
JSLoot by famos0

Looting URLs, IPv4 addresses, Base64 encoded stuff, AWS-keys or whatever you want from JavaScript

-- WHERE TO LOOT ? -- 
 -u, --url <url>                Loot from on the URL
 -f, --file <path>              Loot from a local file
 -d, --directory <path>         Loot from a directory
 -r, --recurse <path>           To combine with -d option. Loot recursively
 -s, --stdin                    Loot from URLs given by Stdin

-- WHAT TO LOOT ? -- 
 -e, --regexp <PATTERNS>        Loot with a custom pattern
 -g, --grep-pattern <1,2,...>   When specified, custom the looting patterns :
                                - 1 : Looting URLs
                                - 2 : Looting AWS Keys
                                - 3 : Looting Base64 artifacts
                                - 4 : Looting IPv4 addresses
                                - 5 : Looting nothing on the default patterns
                                      Can be used in complement with -e

-- SHOW THE LOOT -- 
 -H, --with-filename            Show filename/URL of loot location
 -v, --verbose                  Print with detailed output and colors
```

## Happy Looting :)

## TODO
1. Proxy
2. Header
3. Cookie