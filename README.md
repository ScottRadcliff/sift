# Sift

![Sift — a metal sifter filtering code fragments into matching results](assets/sift-banner.png)

Sift is a self-contained agent skill for searching local text files and exporting matches to CSV. Ask for a search in conversation; the skill runs its bundled Python helper and reports matching file locations, skipped files, and incomplete results.

## Requirements

Python 3.11 or newer must be available to the agent, along with access to the files you want to search. The skill includes its search dependency. No package installation, separate executable, Go, or Docker is required.

## Install

The installable unit is the entire [skills/sift](skills/sift) folder. Keep its `SKILL.md`, `agents/`, `scripts/` (including `_vendor/`), and `references/` together.

For a personal Codex installation, run this from the repository root:

```sh
python3 - <<'PY'
from pathlib import Path
import shutil

source = Path('skills/sift')
destination = Path.home() / '.agents' / 'skills' / 'sift'
shutil.copytree(source, destination, ignore=shutil.ignore_patterns('__pycache__', '*.pyc'))
print(f'Installed Sift at {destination}')
PY
```

The copy refuses an existing destination. To update, review and replace your existing skill folder with the new complete folder. The copied installation works independently of this checkout. The personal discovery location is documented in the [official OpenAI skills guide](https://learn.chatgpt.com/docs/build-skills). If the skill does not appear, restart Codex.

To try the repository copy before installing, tell your agent:

> Use the Sift skill at `skills/sift/SKILL.md` to find payment failures in `samples/`.

## Use Sift

Once installed, invoke it in conversation:

> Use $sift to find payment failures in this project.

> Use $sift to find exactly “Payment” in `app/`, matching case.

> Use $sift to search `logs/` with the regex `payment|retry` and export matches to a new CSV.

> Use $sift to search `src/` for `/etc/hosts`, allowing up to ten seconds.

Search text is literal unless you request a regex. Matching is case-insensitive by default. Directory searches recurse and respect `.gitignore` rules within the selected root. Hidden descendants and binary files are excluded; the skill reports relevant binary skips and search errors. Explicitly named hidden or ignored files remain searchable.

The default search budget is three seconds. A timeout retains available matches and marks the result incomplete. The skill can rerun with a narrower scope or an authorized longer budget; reruns start over.

CSV exports contain `filename`, `line_number`, and original matching-line `text`. Existing exports are never overwritten. Empty results produce a header-only CSV; incomplete searches can leave a partial export, which the skill identifies in its response.

See the [skill instructions](skills/sift/SKILL.md) and [search behavior reference](skills/sift/references/search-behavior.md) for regex semantics, scope exclusions, Unicode handling, result fields, and exit statuses.

## Development

The supported interface is the skill. Its Python arguments are an internal contract for agent calls and development checks.

```text
skills/sift/          Complete installable skill
  SKILL.md           Agent workflow and invocation
  scripts/           Python helper and bundled dependency
  references/        Detailed search behavior
  agents/            Skill UI metadata
samples/             Synthetic files for manual QA
tests/migration/    Behavior and integration checks
prototypes/          Earlier migration experiments, not installed
plans/               Migration decisions and verification history
```

Run checks from the repository root:

```sh
make test
make portable
```

Make is optional; equivalent Python commands and test details are in the [test guide](tests/migration/README.md). The portability check runs a copied skill from an unrelated project with installed packages disabled and an empty executable PATH. This catches accidental dependencies on the checkout or installed tools.

For direct helper checks, use the [sample QA guide](samples/README.md). Helpers always return JSON search results and never prompt; the agent handles presentation and follow-up questions.

[Python CI](.github/workflows/ci.yml) runs the full suite and portable-installation check on Linux with Python 3.11 and 3.13. Local validation has also covered macOS. Windows has not been validated.

The [completed migration record](plans/python-skill-migration.md) explains the transition from the former Go CLI. Old local `bin/`, `dist/`, and coverage artifacts are ignored and are not used by the skill.

## Bundled dependency

The skill includes unmodified pathspec 1.1.1 under MPL-2.0. Its [license](skills/sift/scripts/_vendor/pathspec/LICENSE) and [source attribution](skills/sift/scripts/_vendor/NOTICE.md) travel with the installed skill folder.
