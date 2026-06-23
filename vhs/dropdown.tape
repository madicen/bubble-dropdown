# VHS tape for the basic example
# Run from bubble-dropdown dir: vhs vhs/dropdown.tape

Output "screenshots/dropdown.gif"

Set FontSize 14
Set Width 700
Set Height 400
Set Shell "bash"
Set Theme "Builtin Dark"
Set TypingSpeed 75ms
Env COLORTERM truecolor
Env TERM xterm-256color

Hide
Type "go run ./examples/basic"
Enter
Sleep 1.5s
Show

# Keyboard: Tab to focus fruit, open with Enter, navigate, select
Sleep 1s
Enter
Sleep 1s
Down 2
Sleep 800ms
Down 2
Sleep 800ms
Enter
Sleep 1s

# Tab to color dropdown, cycle with arrows while closed
Tab
Sleep 600ms
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 600ms

# Open color with Space, pick a different item
Space
Sleep 800ms
Up
Sleep 600ms
Up
Sleep 600ms
Enter
Sleep 1s

# Tab to size, open and select
Tab
Sleep 600ms
Enter
Sleep 800ms
Down
Sleep 500ms
Down
Sleep 500ms
Enter
Sleep 1.5s

Type "q"
Sleep 500ms
