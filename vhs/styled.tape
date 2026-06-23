# VHS tape for the styled example
# Run from bubble-dropdown dir: vhs vhs/styled.tape

Output "screenshots/styled.gif"

Set FontSize 14
Set Width 700
Set Height 400
Set Shell "bash"
Set Theme "Builtin Dark"
Set TypingSpeed 75ms
Env COLORTERM truecolor
Env TERM xterm-256color

Hide
Type "go run ./examples/styled"
Enter
Sleep 1.5s
Show

# Violet-themed fruit dropdown: open, navigate, select
Sleep 1s
Enter
Sleep 1s
Down 2
Sleep 800ms
Down 2
Sleep 800ms
Enter
Sleep 1s

# Tab to the green-themed color dropdown, open with Space
Tab
Sleep 600ms
Space
Sleep 1s
Down 3
Sleep 800ms
Up
Sleep 600ms
Enter
Sleep 1.5s

Type "q"
Sleep 500ms
