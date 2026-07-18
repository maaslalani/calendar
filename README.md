# Calendar

Calendar in the terminal.

<img src="demo/demo.gif" width="1000" alt="calendar showing events in the terminal" />

### Usage

```bash
cal
```

> [!NOTE]
> On first launch, allow your terminal to access Calendars in System Settings.

### Installation

Install with Go:

```
go install github.com/maaslalani/calendar@main
```

Or download a binary from the [releases](https://github.com/maaslalani/calendar/releases).

## EventKit

`calendar` reads your events directly from Apple's native
[EventKit](https://developer.apple.com/documentation/eventkit) framework, so it
shows the same calendars as Calendar.app.[^go-eventkit]

[^go-eventkit]: EventKit access is powered by [`go-eventkit`](https://github.com/BRO3886/go-eventkit).

## Calendar File System

> [!WARNING]
> Not Implemented.

Interact with your calendar as a file system.
See [`0001-calfs.md`](0001-calfs.md) for the design.

```
.
└── 2026
    └── 01
        ├── 01
        │   └── Meeting
        │       ├── description
        │       ├── invited
        │       │   ├── alice@example.com
        │       │   ├── bob@example.com
        │       │   └── carol@example.com
        │       ├── start -> 8:00AM
        │       └── end -> 10:00AM
        ├── 02
        ├── 03
        ├── 04
        │   ├── Event
        │   └── Event
        └── 05
```

## License

[MIT](https://github.com/maaslalani/calendar/blob/main/LICENSE)

## Feedback

I'd love to hear your feedback on improving `calendar`.

Feel free to reach out via:
* [Email](mailto:maas@lalani.dev)
* [Twitter](https://twitter.com/maaslalani)
* [GitHub issues](https://github.com/maaslalani/calendar/issues/new)

---

<sub><sub>z</sub></sub><sub>z</sub>z
