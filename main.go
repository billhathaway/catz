package main

// cattz ('cat' + 'tz') reads files or stdin and converts timestamps from one timezone to another
// a common use case would be a log file with timestamps in UTC and converting those to your local timezone

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"time"

	"github.com/billhathaway/strftime"
)

type controller struct {
	// srcLoc is TZ for input
	srcLoc *time.Location
	// destLoc is TZ for output
	destLoc *time.Location
	// format is output format in time.Format() style
	format string
	// re is a regular expression used to search for timestamps
	re *regexp.Regexp
	// buf is a buffer used when re-assembling output
	buf *bytes.Buffer
	// matchLimit is used to control how many times we try to find timestamps in a given line
	// it will either be set to -1 (unlimited) or 1 (only handle first timestamp)
	matchLimit int
	// debug enables debugging output
	debug bool
}

func (c *controller) debugf(format string, v ...interface{}) {
	if c.debug {
		fmt.Printf(format, v...)
	}
}

// map between strftime conversions and regular expression components
var strfmtimeRE = map[byte]string{
	'Y': `\d{4}`,
	'm': `\d{2}`,
	'b': `[A-Z][a-z]{2}`,
	'd': `\d{2}`,
	'a': `[A-Z][a-z]{2}`,
	'e': `\d?\d`,
	'H': `\d{2}`,
	'M': `\d{2}`,
	'I': `\d{2}`,
	'p': `[AP]M`,
	'P': `[ap]m`,
	'r': `\d{2}:\d{2}:\d{2} [AP]M`,
	'S': `\d{2}`,
	'T': `\d{2}:\d{2}:\d{2}`,
	'y': `\d{2}`,
	'z': `(-|\+)\d{4}`,
	'Z': `[A-Za-z_]+`,
	'%': `%`,
}

// strfimeToGo is given a strftime() format and builds a regex for parsing
// %Y-%m-%d would turn into
// \d{4}-\d{2}-\d{2}
// TODO: check for completeness
func (c *controller) strfimeToRE(format string) error {
	regexBuf := &bytes.Buffer{}
	var inPercent bool
	for i := range format {
		ch := format[i]
		if inPercent {
			val, ok := strfmtimeRE[ch]
			if !ok {
				return fmt.Errorf("unsupported conversion char %c", ch)
			}
			regexBuf.WriteString(val)
			inPercent = false
		} else if ch == '%' {
			inPercent = true
		} else {
			// deal with literal
			regexBuf.WriteByte(ch)
		}
	}
	c.debugf("strftime regexString=[%s]\n", regexBuf.String())
	var err error
	c.re, err = regexp.Compile(regexBuf.String())
	return err
}

// replaceTimeOffset is passed a timestamp in a source TZ and dest TZ offset in hours
// currently used only for benchmark purposes
func (c *controller) replaceTimeOffset(token string, hoursOffset int) (string, error) {
	source, err := time.ParseInLocation(c.format, token, c.srcLoc)
	if err != nil {
		return token, err
	}
	return source.Add(time.Duration(hoursOffset) * time.Hour).Format(c.format), nil
}

// replaceTime is passed a timestamp in a source TZ and returns the timestamp in the dest TZ
func (c *controller) replaceTime(token string) (string, error) {
	source, err := time.ParseInLocation(c.format, token, c.srcLoc)
	if err != nil {
		return token, err
	}
	return source.In(c.destLoc).Format(c.format), nil
}

// replaceLine returns the line with all the timestamps converted
func (c *controller) replaceLine(line string) string {
	matches := c.re.FindAllStringIndex(line, c.matchLimit)
	if matches == nil {
		c.debugf("DEBUG: no match for regex in line [%s]\n", line)
		return line
	}
	c.buf.Reset()
	offset := 0
	for i, match := range matches {
		c.debugf("in match %d match=[%v]\n", i, match)
		prefix := line[offset:match[0]]
		c.debugf("prefix=[%s]\n", prefix)
		c.buf.WriteString(prefix)
		token := line[match[0]:match[1]]
		c.debugf("token=[%s]\n", token)
		replaced, err := c.replaceTime(token)
		if err != nil {
			fmt.Printf("ERROR parsing orig=[%s] time: %s\n", token, err)
			return line
		}
		c.buf.WriteString(replaced)
		c.debugf("replaced=[%s]\n", replaced)
		offset = match[1]
	}
	if offset < len(line) {
		c.buf.WriteString(line[offset:])
	}
	return c.buf.String()
}

func (c *controller) parse(r io.Reader) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := s.Text()
		fmt.Println(c.replaceLine(line))
	}
}

func (c *controller) execute() int {
	var errCount int
	if len(flag.Args()) > 0 {
		for _, file := range flag.Args() {
			fh, err := os.Open(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s %s %s\n", os.Args[0], file, err)
				errCount++
				continue
			}
			c.parse(fh)
			fh.Close()
		}
		return errCount
	}
	c.parse(os.Stdin)
	return 0
}

func main() {
	tz := "US/Pacific"
	if cattz, ok := os.LookupEnv("CATZ"); ok {
		tz = cattz
	} else {
		stdtz, ok := os.LookupEnv("TZ")
		if ok {
			tz = stdtz
		}
	}
	srcTZ := flag.String("srctz", "UTC", "input time zone")
	destTZ := flag.String("outtz", tz, "output time zone (defaults to $CATZ or $TZ env if available)")
	debug := flag.Bool("d", false, "enable debug logging")
	timeFormat := flag.String("t", "%Y-%m-%d %H", "strftime format")
	first := flag.Bool("first", false, "only replace first timestamp match per line")
	flag.Parse()

	sourceLocation, err := time.LoadLocation(*srcTZ)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR input TZ %s not known\n", *srcTZ)
		os.Exit(1)
	}

	destLocation, err := time.LoadLocation(*destTZ)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR output TZ %s not known\n", *destTZ)
		os.Exit(1)
	}
	c := &controller{
		srcLoc:  sourceLocation,
		destLoc: destLocation,
		buf:     &bytes.Buffer{},
		debug:   *debug,
	}
	// verify we know how to handle the time format
	c.format, err = strftime.New(*timeFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR time format [%s] had problem converting to go format%s\n", *timeFormat, err)
		os.Exit(1)
	}
	err = c.strfimeToRE(*timeFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR time format [%s] had problem to regex %s\n", *timeFormat, err)
		os.Exit(1)
	}
	// only replace first timestamp occurance, or replace all
	if *first {
		c.matchLimit = 1
	} else {
		c.matchLimit = -1
	}

	os.Exit(c.execute())
}
