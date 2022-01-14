# gott - go time tracker

a `timewarrior` alternative that fits my needs.

**why not timewarrior?**

- in `timew` i cannot add a time for a specific day while ignoring the timespan. I want to just add X hours to it without careing about the conrete time.
- in `timew` i cannot add project or reference information from `taskwarrior`. The annotation method is somehow weird.

## Usage

### `start`
You can just begin with tracking your time by starting the tracking:

```bash
$ gott start [add your message here [project:projectname] [+tag01 +tag02] [ref:EXTERNAL_ID]
```

Example:

```bash
$ gott start writing documentation for gott project:gott.docs +docs ref:ID-1337
tracking writing documentation for gott -- proj:gott.docs -- docs -- ref:ID-1337
    Started          01-14 22:44
    Current (mins)   00:00
    Total   (today)  00:01
```

The status shows the current tracking, including project, tags and reference. It also shows when the current tracking started, the currently trackt timespan and the summed up timespan for the current day.

### `annotate`

If you want to add some annotation to the running interval, use the `annotate` subcommand:

```bash
$ gott annotate +another-tag
tracking writing documentation for gott -- proj:gott.docs -- docs, another-tag -- ref:ID-1337
    Started          01-14 22:44
    Current (mins)   00:00
    Total   (today)  00:tag
```

### `stop`

To stop the current interval use the `stop` subcommand. It shows the status of the stopped, just collected interval.

```bash
$ gott stop
tracking writing documentation for gott -- proj:gott.docs -- docs, another-tag -- ref:ID-1337
    Started          01-14 22:44
    Stopped          01-14 22:44
    Current (mins)   00:05
    Total   (today)  00:06
```

### `continue`

To restart work on the latest task you can just restart an interval with the same config. You can use the `continue` subcommand for this.

```bash
$ gott continue
tracking writing documentation for gott -- proj:gott.docs -- docs, another-tag -- ref:ID-1337
    Started          01-14 22:50
    Current (mins)   00:00
    Total   (today)  00:06
```

### `cancel`

If you started an interval by mistake or have another reason to cancel some interval and discard the current interval you can use the `cancel` subcommand.

```bash
$ gott cancel
```

### `summary`

The summary command prints out the current collection state. By default it only prints the today's collected intervals. You can change this by filtering with the keywords you remember from Taskwarror: `:today`, `:yesterday`, `:week`, `:month`, `:all` or a date filter with `YYYY-MM-DD`.

```
$ gott summary :today
CWEEK  DAY    BEGIN  END    DURATION  PROJECT  TAG                ANNOTATION
-----  ---    -----  ---    --------  -------  ---                ----------
2      01-14  22:44  22:50  00:06     gott     docs, another-tag  writing gott documentation
              22:55  23:00  00:05     gott     docs, another-tag  writing gott documentation
                     day =  00:11
              wk =          00:11°

```

### `track`

To add a missing interval to a given day. You can use the `track` subcommand for this.

```bash
$ go track 2022-11-20 3h -- bake a cake 
```

Magic names can be used here, too:


```bash
$ go track :today 3h -- bake a cake 
$ go track :yesterday 3h -- bake a bread
```

### `edit`

If you want to bulk edit some interval you can use the `edit` subcommand. It exports the given filter to a text file and opens it up in your `$EDITOR`. When closing the changes become applied in bulk.

```bash
$ go edit :today
```

## Configuration

`gott` uses viper for configuration management. With its help it checks your `$HOME` and the folder along the `gott` binary for a  `.gottrc` file with the possible endings: `ini`, `json` or `yml`.



| *Configkey* | *Description* |
|-------------|---------------|
| `databasename` | The name and location of the database file. |



