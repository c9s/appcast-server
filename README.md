Appcast Server implemented in Go
================================

Build
-----

	go build -x && ./appcast-server --domain appcast.yourhost.com --bind :5000

Create New Channel
------------------

	/channel/create

Upload New Release
------------------

	/release/upload/{channel id}/{channel token}

Get AppCast XML
---------------

	/appcast/{channel id}/{channel token}
