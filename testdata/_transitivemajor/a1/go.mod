module foo.bar/a

go 1.25

require (
	foo.bar/d/v2 v2.0.0
)

replace foo.bar/d/v2 => ../d1
