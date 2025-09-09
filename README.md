[![](https://img.shields.io/badge/nix-flake-blue?style=for-the-badge&logo=nixos&logoColor=white&logoSize=auto)](#)
[![](https://img.shields.io/github/actions/workflow/status/krostar/git-workhours/quality.yml?branch=main&style=for-the-badge)](#)
[![](https://img.shields.io/github/go-mod/go-version/krostar/git-workhours?label=go&style=for-the-badge)](#)
[![](https://img.shields.io/github/v/tag/krostar/git-workhours?sort=semver&style=for-the-badge)](https://github.com/krostar/git-workhours/tags)
[![](https://img.shields.io/github/license/krostar/git-workhours?style=for-the-badge)](#)

# git-workhours

Git hooks to help maintain healthy work-life boundaries by managing when commits happen — or at least how they appear in your Git history.

## What it does

`git-workhours` is composed of few git hooks that can:

- **Validate commits** – Warn or block commits outside configured work hours (`pre-commit` / `pre-push`)
- **Adjust timestamps** – Automatically rewrite commit dates to fall within work hours (`post-commit`)

## Why it matters

Your Git history is more than just code — it’s a permanent activity log. Commit timestamps can unintentionally expose:

- **Your personal routines or timezone**
- **Late-night coding sessions**
- **Weekend or holiday work**

### Professional impact

- **Managers & reviews** – Commit timing can be read (fairly or not) as productivity signals
- **Colleagues** – Visible after-hours activity risks setting unhealthy norms
- **Privacy Concerns** - Work patterns, lifestyle, and even geography can be inferred from commit history

## Installation

```bash
go install github.com/krostar/git-workhours@latest
```

## Configuration

You can customize all settings via git configuration (*`wh.*` section*), environment (*prefixed with `GIT_WORKHOURS_`*), and flags (*use `--help` to list them all*).

### Schedule format

The git configuration `wh.schedule`, env **`GIT_WORKHOURS_SCHEDULE`**, or flag `--schedule`, defines your weekly work schedule.

- **Format**: comma-separated list of 7 entries (one per day), **Sunday to Saturday**.
- **Empty day**: nothing to specify if no work is scheduled.
- **Multiple shifts per day**: separate them with `+`.
- **Shift format**: `start-end`, where both `start` and `end` are `time.Duration` values (e.g. `9h`, `9h30m`, `17h45m`).

#### Examples

- **Standard 9–5, weekdays only**: `,9h-17h,9h-17h,9h-17h,9h-17h,9h-17h,` → No work on Sunday/Saturday, 9–17 on Mon–Fri.
- **Split shifts (morning + afternoon), weekdays**: `,9h-12h+13h-17h,9h-12h+13h-17h,9h-12h+13h-17h,9h-12h+13h-17h,9h-12h+13h-17h,` → Typical office schedule with lunch breaks.
- **Flexible & irregular**: `14h-18h,10h-13h+14h-19h,,,8h-12h,,20h-23h` → Sunday afternoon work, Monday with two shifts, Thursday morning only, Saturday night coding.

### Invert schedule

The git configuration `wh.invertschedule`, env **`GIT_WORKHOURS_INVERT_SCHEDULE`**, or flag `--invert-schedule`, invert the configured work schedule.
The defined schedule becomes the **blocked hours**, all other times are considered **valid work hours**.

This is useful if you want to *explicitly block only certain hours* (e.g., nights, weekends, or personal time) instead of defining every possible valid shift.

#### Examples

- **Block weekends entirely**: `0h-24h,,,,,,0h-24h` → Sunday and Saturday are fully blocked, commits allowed only Mon–Fri.

### Allow overtime

The git configuration `wh.allowovertime`, env **`GIT_WORKHOURS_ALLOW_OVERTIME`**, or flag `--allow-overtime`, displays warning instead of failure when working overtime.

### Fake valid time

The git configuration `wh.fakevalidtime`, env **`GIT_WORKHOURS_FAKE_VALID_TIME`**, or flag `--fake-valid-time`, fixes git commit time when working overtime, requires allowing overtime.
Only used in `post-commit` hook.

## Usage

### Manually

Create hook in your project's dir: `.git/hooks/{pre-commit,post-commit,pre-push`, and `chmod +x` them.

Create a simple script file executing git-workhours, like for pre-commit:

```bash
#!/usr/bin/env sh
git-workhours hooks pre-commit
```

If you want to do it for all projects, set the git hooks directory in your git config

```ini
// git config
[core]
    hooksPath = "$HOME/.local/share/git/hooks/"
```

### Nix

The flake exposes git-workhours package, and a git-workhours module to use within home-manager.

```nix
{
  programs.git = {
    user.email = "me@my.company";
    wh.schedule = ",9h-19h,9h-19h,9h-19h,9h-19h,9h-19h,";

    includes = [
      {
        condition = "gitdir:~/Personal/";
        contents = {
          user.email = "krostar@users.noreply.github.com";
          wh.invertschedule = true;
        };
      }
    ];
  };
}
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.
