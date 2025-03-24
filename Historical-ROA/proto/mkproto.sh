#/bin/sh
/usr/bin/protoc rarc.proto --proto_path=. --go_out=.  --go_opt=paths=source_relative

