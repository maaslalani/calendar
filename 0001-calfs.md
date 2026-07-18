# CalFS

- **Status:** Draft
- **Version:** 0.1
- **Author:** Maas Lalani
- **Created:** 2026-07-17

Interact with your calendar as a file system.

## Concepts

Your calendar is a file system with directories and files.

The folder hierarchy:

```
.
├── 2026
│   ├── 01
│   │   ├── 01
│   │   │   └── Event
│   │   ├── 02
│   │   ├── 03
│   │   ├── 04
│   │   │   ├── Event
│   │   │   └── Event
│   │   └── 05
```

## Symbolic links

* `today`
* `tomorrow`
* `yesterday`
* `month`
* `week/sunday`
* `week/monday`
* `week/tuesday`
* `week/wednesday`
* `week/thursday`
* `week/friday`
* `week/saturday`

## Interaction

### List events for a specific date

`ls` the directory representing the date.

```
ls 2021/01/01
```

### List events for `today`

`ls` the directory representing `today`, which is a `symlink` to the directory
representing today's date.

```
ls today
```

### Create a new event

Create a directory with the name being the event name.

```
mkdir tomorrow/Meeting
```

## Event Directory

A meeting is a directory.

```
Meeting
├─ description
├─ invited/
├─ start -> 8:00AM
└─ end -> 10:00AM
```

* `description` is a file with the meeting description.
* `invited` is a directory with files named the emails of the guests invited.
* `start` is a symlink to a `time`.
* `end` is a symlink to a `time`.

A `time` is a file located within the `calendar/time/` directory. It
represents times with intervals of `x` minutes.
