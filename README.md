catz
--
catz is similar to the 'cat' command that displays the contents of a file or files (or stdin), except that it can also replace
time/datestamps in a specific format with the times re-written for a different time zone.  The name is a portmanteau of 'cat' + 'tz'

The original target was humans looking at log files that were generated on a server that was set to UTC and then trying to figure out what time the events occurred in the local time zone.

catz options:

    -t string
      strftime format (default "%Y-%m-%d %H")

    -outtz string
       output time zone (defaults to $CATZ or $TZ env if available) (default "US/Pacific")

    -srctz string
       input time zone (default "UTC")

    -first
       only replace first timestamp match per line (default all that match)


Example: if you had a file with some lines written in UTC such as:

    cat sample.txt
    2016-01-18 04:44:35,883 INFO path=/ method=GET status=200
    2016-01-18 06:23:19,967 INFO path=/users method=POST status=200
    2016-01-18 11:04:26,076 INFO path=/instances method=GET status=200

And then used catz to view the file, you would see the times have been re-written to US/Pacific TZ:

    catz sample.txt
    2016-01-17 20:44:35,883 INFO path=/ method=GET status=200
    2016-01-17 22:23:19,967 INFO path=/users method=POST status=200
    2016-01-18 03:04:26,076 INFO path=/instances method=GET status=200

If you are working with a log using a different time format, such as nginx, which uses default timestamps in a pattern like '11/Jan/2014:18:00:00', you could use:  

    catz -t "%d/%b/%Y:%H" access.log

It typically isn't necessary to have the minutes/seconds/milliseconds searched and replaced since those values would stay the same.
