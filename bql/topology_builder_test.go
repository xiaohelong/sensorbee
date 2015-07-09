package bql

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestCreateSourceStmt(t *testing.T) {
	Convey("Given a BQL TopologyBuilder", t, func() {
		dt := newTestTopology()
		Reset(func() {
			dt.Stop()
		})
		tb, err := NewTopologyBuilder(dt)
		So(err, ShouldBeNil)

		Convey("When running CREATE SOURCE with a dummy source", func() {
			err := addBQLToTopology(tb, `CREATE SOURCE hoge TYPE dummy`)

			Convey("Then there should be no error", func() {
				So(err, ShouldBeNil)
			})

			Convey("And when running another CREATE SOURCE using the same name", func() {
				err := addBQLToTopology(tb, `CREATE SOURCE hoge TYPE dummy`)

				Convey("Then an error should be returned", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "already")
				})
			})

			Convey("And when running another CREATE SOURCE using a different name", func() {
				err := addBQLToTopology(tb, `CREATE SOURCE src TYPE dummy`)

				Convey("Then there should be no error", func() {
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("When running CREATE SOURCE with valid parameters", func() {
			err := addBQLToTopology(tb, `CREATE SOURCE hoge TYPE dummy WITH num=4`)

			Convey("Then there should be no error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When running CREATE SOURCE with invalid parameters", func() {
			err := addBQLToTopology(tb, `CREATE SOURCE hoge TYPE dummy WITH num='bar'`)

			Convey("Then an error should be returned", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "num: cannot convert value \"bar\" into integer")
			})
		})

		Convey("When running CREATE SOURCE with unknown parameters", func() {
			err := addBQLToTopology(tb, `CREATE SOURCE hoge TYPE dummy WITH foo='bar'`)

			Convey("Then an error should be returned", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "unknown source parameter: foo")
			})
		})

		Convey("When running CREATE SOURCE with an unknown source type", func() {
			err := addBQLToTopology(tb, `CREATE SOURCE hoge TYPE foo`)

			Convey("Then an error should be returned", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "not registered")
			})
		})
	})
}

func TestCreateStreamAsSelectStmt(t *testing.T) {
	Convey("Given a BQL TopologyBuilder with source and stream", t, func() {
		dt := newTestTopology()
		Reset(func() {
			dt.Stop()
		})
		tb, err := NewTopologyBuilder(dt)
		So(err, ShouldBeNil)
		err = addBQLToTopology(tb, `CREATE PAUSED SOURCE s TYPE dummy`)
		So(err, ShouldBeNil)

		Convey("When running CREATE STREAM AS SELECT on an existing stream", func() {
			err := addBQLToTopology(tb, `CREATE STREAM t AS SELECT ISTREAM int FROM
                s [RANGE 2 SECONDS] WHERE int=2`)

			Convey("Then there should be no error", func() {
				So(err, ShouldBeNil)
			})

			Convey("And when joining source and stream", func() {
				err := addBQLToTopology(tb, `CREATE STREAM x AS SELECT ISTREAM s:int FROM
                s [RANGE 2 SECONDS], t [RANGE 3 TUPLES]`)

				Convey("Then there should be no error", func() {
					So(err, ShouldBeNil)
				})
			})

			Convey("And when self-joining the stream", func() {
				err := addBQLToTopology(tb, `CREATE STREAM x AS SELECT ISTREAM s:int FROM
                t [RANGE 2 SECONDS] AS s, t [RANGE 3 TUPLES]`)

				Convey("Then there should be no error", func() {
					So(err, ShouldBeNil)
				})
			})

			Convey("And when running another CREATE STREAM AS SELECT with the same name", func() {
				err := addBQLToTopology(tb, `CREATE STREAM t AS SELECT ISTREAM int FROM
                s [RANGE 1 TUPLES] WHERE int=1`)

				Convey("Then an error should be returned the second time", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "already")
				})
			})

			Convey("And when running another CREATE STREAM AS SELECT with a different name", func() {
				err := addBQLToTopology(tb, `CREATE STREAM u AS SELECT ISTREAM int FROM
                s [RANGE 1 TUPLES] WHERE int=1`)

				Convey("Then there should be no error", func() {
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("When running CREATE STREAM AS SELECT with a UDSF", func() {
			Convey("If all parameters are foldable", func() {
				err := addBQLToTopology(tb, `CREATE STREAM t AS SELECT ISTREAM int FROM
                duplicate('s', 2) [RANGE 2 SECONDS] WHERE int=2`)

				Convey("Then there should be no error", func() {
					So(err, ShouldBeNil)
				})

				Convey("And when joining source and UDSF", func() {
					err := addBQLToTopology(tb, `CREATE STREAM x AS SELECT ISTREAM s:int FROM
                s [RANGE 2 SECONDS], duplicate('s', 2) [RANGE 3 TUPLES]`)

					Convey("Then there should be no error", func() {
						So(err, ShouldBeNil)
					})
				})

				Convey("And when self-joining the UDSF", func() {
					err := addBQLToTopology(tb, `CREATE STREAM x AS SELECT ISTREAM s:int FROM
                duplicate('s', 2) [RANGE 2 SECONDS] AS s, duplicate('s', 4) [RANGE 3 TUPLES] AS t`)

					Convey("Then there should be no error", func() {
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("If not all parameters are foldable", func() {
				err := addBQLToTopology(tb, `CREATE STREAM t AS SELECT ISTREAM int FROM
                duplicate('s', int) [RANGE 2 SECONDS] WHERE int=2`)

				Convey("Then there should be an error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "not foldable")
				})
			})

			Convey("If the UDSF is called with the wrong number of arguments", func() {
				err := addBQLToTopology(tb, `CREATE STREAM t AS SELECT ISTREAM int FROM
                duplicate('s', 1, 2) [RANGE 2 SECONDS] WHERE int=2`)

				Convey("Then there should be an error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "arity")
				})
			})

			Convey("If the UDSF doesn't depend on an input", func() {
				err := addBQLToTopology(tb, `CREATE STREAM t AS SELECT ISTREAM int FROM
                no_input_duplicate('s', 2) [RANGE 2 SECONDS] WHERE int=2`)

				Convey("Then there should be an error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "at least")
				})
			})

			Convey("If creating the UDSF fails", func() {
				err := addBQLToTopology(tb, `CREATE STREAM t AS SELECT ISTREAM int FROM
                failing_duplicate('s', 2) [RANGE 2 SECONDS] WHERE int=2`)

				Convey("Then there should be an error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "test UDSF creation failed")
				})
			})

			Convey("If the UDSF isn't registered", func() {
				err := addBQLToTopology(tb, `CREATE STREAM t AS SELECT ISTREAM int FROM
                serial(1, 5) [RANGE 2 SECONDS] WHERE int=2`)

				Convey("Then there should be an error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "not registered")
				})
			})
		})

		Convey("When running CREATE STREAM AS SELECT on a non-existing stream", func() {
			err := addBQLToTopology(tb, `CREATE STREAM t AS SELECT ISTREAM int FROM
                bar [RANGE 2 SECONDS] WHERE int=2`)

			Convey("Then an error should be returned", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "was not found")
			})
		})

		Convey("When running CREATE STREAM AS SELECT with input and output name the same", func() {
			err := addBQLToTopology(tb, `CREATE STREAM s AS SELECT ISTREAM int FROM
                s [RANGE 2 SECONDS] WHERE int=2`)

			Convey("Then an error should be returned", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "already")
			})
		})
	})
}

func TestCreateSinkStmt(t *testing.T) {
	Convey("Given a BQL TopologyBuilder", t, func() {
		dt := newTestTopology()
		Reset(func() {
			dt.Stop()
		})
		tb, err := NewTopologyBuilder(dt)
		So(err, ShouldBeNil)

		Convey("When running CREATE SINK with a collector source", func() {
			err := addBQLToTopology(tb, `CREATE SINK hoge TYPE collector`)

			Convey("Then there should be no error", func() {
				So(err, ShouldBeNil)
			})

			Convey("And when running another CREATE SINK using the same name", func() {
				err := addBQLToTopology(tb, `CREATE SINK hoge TYPE collector`)

				Convey("Then an error should be returned", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "already")
				})
			})
		})
		Convey("When running CREATE SINK with invalid parameters", func() {
			err := addBQLToTopology(tb, `CREATE SINK hoge TYPE collector WITH foo='bar'`)

			Convey("Then an error should be returned", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "unknown sink parameter: foo")
			})
		})
		Convey("When running CREATE SINK with an unknown sink type", func() {
			err := addBQLToTopology(tb, `CREATE SINK hoge TYPE foo`)

			Convey("Then an error should be returned", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "not registered")
			})
		})
	})
}

func TestInsertIntoSelectStmt(t *testing.T) {
	Convey("Given a BQL TopologyBuilder with source, stream, and sink", t, func() {
		dt := newTestTopology()
		Reset(func() {
			dt.Stop()
		})
		tb, err := NewTopologyBuilder(dt)
		So(err, ShouldBeNil)
		err = addBQLToTopology(tb, `CREATE PAUSED SOURCE s TYPE dummy`)
		So(err, ShouldBeNil)
		err = addBQLToTopology(tb, `CREATE STREAM t AS SELECT RSTREAM * FROM s [RANGE 1 TUPLES]`)
		So(err, ShouldBeNil)
		err = addBQLToTopology(tb, `CREATE SINK foo TYPE collector`)
		So(err, ShouldBeNil)

		Convey("When running INSERT INTO SELECT on an existing source and sink", func() {
			err := addBQLToTopology(tb, `INSERT INTO foo SELECT RSTREAM int FROM s [RANGE 1 TUPLES]`)

			Convey("Then there should be no error", func() {
				So(err, ShouldBeNil)
			})

			Convey("And when running another INSERT INTO SELECT", func() {
				err := addBQLToTopology(tb, `INSERT INTO foo SELECT RSTREAM int FROM t [RANGE 1 TUPLES]`)

				Convey("Then there should be no error", func() {
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("When running INSERT INTO SELECT on a non-existing stream", func() {
			err := addBQLToTopology(tb, `INSERT INTO foo SELECT RSTREAM int FROM foo [RANGE 1 TUPLES]`)

			Convey("Then an error should be returned", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "was not found")
			})
		})

		Convey("When running INSERT INTO SELECT on a non-existing sink", func() {
			err := addBQLToTopology(tb, `INSERT INTO bar SELECT RSTREAM int FROM s [RANGE 1 TUPLES]`)

			Convey("Then an error should be returned", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "was not found")
			})
		})
	})
}

func TestInsertIntoFromStmt(t *testing.T) {
	Convey("Given a BQL TopologyBuilder with source, stream, and sink", t, func() {
		dt := newTestTopology()
		Reset(func() {
			dt.Stop()
		})
		tb, err := NewTopologyBuilder(dt)
		So(err, ShouldBeNil)
		err = addBQLToTopology(tb, `CREATE PAUSED SOURCE s TYPE dummy`)
		So(err, ShouldBeNil)
		err = addBQLToTopology(tb, `CREATE STREAM t AS SELECT RSTREAM * FROM s [RANGE 1 TUPLES]`)
		So(err, ShouldBeNil)
		err = addBQLToTopology(tb, `CREATE SINK foo TYPE collector`)
		So(err, ShouldBeNil)

		Convey("When running INSERT INTO on an existing source and sink", func() {
			err := addBQLToTopology(tb, `INSERT INTO foo FROM s`)

			Convey("Then there should be no error", func() {
				So(err, ShouldBeNil)
			})

			Convey("And when running another INSERT INTO", func() {
				err := addBQLToTopology(tb, `INSERT INTO foo FROM t`)

				Convey("Then there should be no error", func() {
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("When running INSERT INTO on a non-existing stream", func() {
			err := addBQLToTopology(tb, `INSERT INTO foo FROM foo`)

			Convey("Then an error should be returned", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "was not found")
			})
		})

		Convey("When running INSERT INTO on a non-existing sink", func() {
			err := addBQLToTopology(tb, `INSERT INTO bar FROM s`)

			Convey("Then an error should be returned", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "was not found")
			})
		})
	})
}

func TestInsertEquivalence(t *testing.T) {
	Convey("Given a BQL TopologyBuilder with source and two sinks", t, func() {
		dt := newTestTopology()
		Reset(func() {
			dt.Stop()
		})
		tb, err := NewTopologyBuilder(dt)
		So(err, ShouldBeNil)
		err = addBQLToTopology(tb, `CREATE PAUSED SOURCE s TYPE dummy WITH num=4`)
		So(err, ShouldBeNil)
		err = addBQLToTopology(tb, `CREATE SINK foo TYPE collector`)
		So(err, ShouldBeNil)
		_foo, err := dt.Sink("foo")
		So(err, ShouldBeNil)
		foo := _foo.Sink().(*tupleCollectorSink)
		err = addBQLToTopology(tb, `CREATE SINK bar TYPE collector`)
		So(err, ShouldBeNil)
		_bar, err := dt.Sink("bar")
		So(err, ShouldBeNil)
		bar := _bar.Sink().(*tupleCollectorSink)

		Convey("When one source is connected directly and one indirectly", func() {
			err := addBQLToTopology(tb, `INSERT INTO foo FROM s`)
			So(err, ShouldBeNil)

			err = addBQLToTopology(tb, `INSERT INTO bar SELECT RSTREAM * FROM s [RANGE 1 TUPLES]`)
			So(err, ShouldBeNil)

			Convey("Then the results should be the same", func() {
				err = addBQLToTopology(tb, `RESUME SOURCE s`)
				So(err, ShouldBeNil)

				foo.Wait(4)
				bar.Wait(4)
				for i := 0; i < 4; i++ {
					// they go different paths, but everything else should be the same
					foo.Tuples[i].Trace = nil
					bar.Tuples[i].Trace = nil
					So(foo.Tuples[i], ShouldResemble, bar.Tuples[i])
				}
			})
		})
	})
}

func TestMultipleStatements(t *testing.T) {
	Convey("Given an empty BQL TopologyBuilder", t, func() {
		dt := newTestTopology()
		Reset(func() {
			dt.Stop()
		})
		tb, err := NewTopologyBuilder(dt)
		So(err, ShouldBeNil)

		Convey("When issuing multiple commands in a good order", func() {
			stmts := `
			CREATE PAUSED SOURCE source TYPE dummy WITH num=4;
			CREATE STREAM box AS SELECT
			  ISTREAM int, str((int+1) % 3) AS x FROM source
			  [RANGE 2 SECONDS] WHERE int % 2 = 0;
			CREATE SINK snk TYPE collector;
			INSERT INTO snk FROM box;
			RESUME SOURCE source;
			`
			err := addBQLToTopology(tb, stmts)

			Convey("Then setup should succeed", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When issuing multiple commands in a bad order", func() {
			stmts := `
			CREATE PAUSED SOURCE source TYPE dummy WITH num=4;
			CREATE STREAM box AS SELECT
			  ISTREAM int, str((int+1) % 3) AS x FROM source
			  [RANGE 2 SECONDS] WHERE int % 2 = 0;
			INSERT INTO snk FROM box;
			CREATE SINK snk TYPE collector;
			RESUME SOURCE source;
			`
			err := addBQLToTopology(tb, stmts)

			Convey("Then setup should fail", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func TestCreateStateStmt(t *testing.T) {
	Convey("Given a BQL TopologyBuilder", t, func() {
		dt := newTestTopology()
		Reset(func() {
			dt.Stop()
		})
		tb, err := NewTopologyBuilder(dt)
		So(err, ShouldBeNil)

		Convey("When creating a dummy UDS", func() {
			So(addBQLToTopology(tb, `CREATE STATE hoge TYPE dummy_uds WITH num=5;`), ShouldBeNil)

			Convey("Then the topology should have the state", func() {
				s, err := dt.Context().SharedStates.Get("hoge")
				So(err, ShouldBeNil)

				Convey("And it should be a dummy UDS having right parameters", func() {
					ds, ok := s.(*dummyUDS)
					So(ok, ShouldBeTrue)
					So(ds.num, ShouldEqual, 5)
				})
			})
		})
	})
}
