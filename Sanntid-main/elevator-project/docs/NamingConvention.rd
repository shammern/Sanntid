From lectures
variables (currentFloor), 
constants (MAX_FLOOR),
global variables (g_elevatorState),
member variables (m_elevatorState),
types (ElevatorDirection),
classes (Order), 
objects (order),
enumerations (ElevatorDirection_Up), 
pointers (pElevatorConfig), 
in parameters (_floor)

From GPT
Golang Naming Conventions 🦫
1. Variables
    Use camelCase for local and package-level variables.
    Keep names short and meaningful.
    Single-letter names (i, j, k) for loop iterators.
2. Constants
    Use PascalCase for exported constants.
    Use UPPER_CASE_SNAKE for package-level constants.
3. Functions
    PascalCase for exported functions.
    camelCase for unexported functions.
    Use verb-noun naming (GetUser(), processOrder()).
4. Structs & Interfaces
    PascalCase for struct and interface names.
    Struct fields: PascalCase (exported), camelCase (unexported).
    Interfaces describing behavior should end with -er (e.g., Reader).
5. Methods
    Follow function naming conventions.
    Receiver variable should be short and meaningful (e.g., u *User).
6. Packages
    Lowercase single words (auth, not AuthSystem).
    No underscores or mixed caps.
7. Error Handling
    Use err or ErrSomething for errors.
    Error messages should be lowercase, no punctuation.
8. Acronyms & Initialisms
    Keep entire acronyms in consistent case (HTTPServer, GetURL).
9. Loop & Temporary Variables
    Use short names (i, j, k) for counters.
    Use ok for map lookups and type assertions.
10. File Naming
    Use lowercase names (auth.go).
    Use snake_case for test files (auth_test.go).