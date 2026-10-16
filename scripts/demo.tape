# Reproducible README walkthrough. Run through scripts/render-demo.sh so
# XDG_CONFIG_HOME is isolated and the saved query always gets history ID 1.
Output dist/demo-base.mp4

Require bash

Set Shell "bash"
Set FontFamily "Menlo"
Set FontSize 18
Set Width 1280
Set Height 720
Set Framerate 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 35ms

Type "./bin/qbx examples/services.json"
Enter
Sleep 2.5s
Down
Sleep 500ms

# Build: iterate the services array.
Type "?"
Sleep 400ms
Type "/Iterate elements"
Enter
Sleep 2.5s

# Filter on the prefilled language value from the first service.
Right@250ms 4
Sleep 400ms
Type "f"
Sleep 1s
Enter
Sleep 2.5s

# Project only the name and replica count.
Type "?"
Type "/Select multiple"
Enter
Sleep 700ms
Down
Space
Down
Space
Sleep 500ms
Enter
Sleep 2.5s

# Sorting a stream visibly adds Collect before Sort.
Type "o"
Sleep 600ms
Type "/replicas"
Tab
Right
Sleep 400ms
Enter
Sleep 800ms
Tab
Right@250ms 2
Down@250ms 3
Right
Tab
Sleep 1.5s
Type "]]"
Sleep 2s
Type "[["
Sleep 1.5s

# Save and show the query in history.
Type "s"
Sleep 500ms
Type "Python services by replicas"
Enter
Sleep 2s
Ctrl+H
Sleep 2.5s
Escape
Sleep 800ms
Type "q"
Sleep 1s

# Reopen the same editable query against a different JSON file.
Type "./bin/qbx history open 1 examples/services-staging.json"
Enter
Sleep 1.5s
Tab
Right@250ms 2
Down@250ms 3
Right
Tab
Sleep 4s
