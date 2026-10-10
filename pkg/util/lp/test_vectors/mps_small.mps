* Retained floating MPS integration fixture
NAME SMALL
ROWS
 N COST
 L R0
 L R1
COLUMNS
    x         COST      3
    x         R0        1
    x         R1        2
    y         COST      2
    y         R0        1
    y         R1        1
    f         COST      5
    f         R0        1
RHS
    RHS1      R0        8
    RHS1      R1        8
BOUNDS
 UP BND       x 4
 UP BND       y 5
 FX BND       f 2
RANGES
    RNG       R0        5
ENDATA
