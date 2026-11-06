"""Evaluate scoped pathspec rules against migration fixtures; not production.

The evaluator supplies pathspec from a temporary wheel via PYTHONPATH. There is
no runtime downloader or dependency on Git. Permission/error handling belongs
to the production traversal, not this selection experiment.
"""

from pathlib import Path

from pathspec import GitIgnoreSpec


def selected_files(root, no_ignore=False):
    root = Path(root)
    if root.is_file():
        return [root]
    selected = []

    def visit(directory, inherited):
        rules = list(inherited)
        ignore = directory / ".gitignore"
        if not no_ignore and ignore.is_file():
            rules.append((directory, GitIgnoreSpec.from_lines(
                ignore.read_text(encoding="utf-8").splitlines(), backend="simple")))
        for path in sorted(directory.iterdir()):
            if path.name.startswith(".") or path.is_symlink():
                continue
            is_directory = path.is_dir()
            ignored = False
            for base, spec in rules:
                relative = path.relative_to(base).as_posix() + ("/" if is_directory else "")
                decision = spec.check_file(relative).include
                if decision is not None:
                    ignored = decision
            if ignored:
                continue
            if is_directory:
                visit(path, rules)
            elif path.is_file():
                selected.append(path)

    visit(root, [])
    return selected
