# subdomainExtractor

A tool to extract subdomains of a given domain from a URL or a list of URLs with the ability to run multiple threads and limit requests.

# Usage Examples:

`subdomainExtractor -d <domain> -u <url> / -f <file>`

### Options Flags
```
subdomainExtractor:
  -d string
    	Target domain to search for subdomains (e.g. ford.com)
  -f string
    	File containing newline-separated URLs to fetch
  -i	Skip TLS certificate verification (use with caution) (default true)
  -o string
    	File to write found subdomains (default: stdout)
  -rps int
    	Maximum number of HTTP requests per second (default 20)
  -t int
    	Maximum number of concurrent threads (default 10)
  -u string
    	URL to fetch and search

```