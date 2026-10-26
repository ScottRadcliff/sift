# Sift is mini grep utility. It contains a small portion of what grep has, and can be called from an agent. It produces results similar to regular grep, with a few changes.


This should be built as a skill.

1. When displaying results, string matches are highlighted
2. Matches are formatted in `line_number:filename match` format.
3. Each result is clickable and loads that file, in that position, in the default editor
4. The UI is `sift 'search' filename`
5. This does not use flags, but has options via either text or questions. Listed below.
6. Has option to export to CSV
7. Does not search binary files, but display the files it couldn't search in the results


## Options

- If the filename is a directory, automatically search recursively, unless it's determined that it can't produce results in less that 3 seconds. If it's over 3 seconds, warn that it's a large directory and ask if the user wants to continue.
- Assume all search are case insenstive, unless told otherwise
- If words are found that are close, but likely misspelled, ask the user if they would like to include those in the results
- Accept regular expressions as both `/term/` with modifiers.
- Warn in the regex is invalid and offer suggestions
 


