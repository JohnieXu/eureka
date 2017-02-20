# ENCRYPT FILE WITH AES-GCM

I couldn't encrypt a file with AES-GCM using the OpenSSL command line tool, so I made this.

Use it at your own risks. (Inb4 someone accuses me of killing innocent people.)

## Install

[Get it here](releases/tag/0.1.0) or build it yourself.

## Usage

`./EncryptFileWithAESGCM -encrypt -file [your-file]` will encrypt a file AND give you a one-time 256-bit AES key.

You're supposed to upload that file somewhere, and send the key to your recipient in a separate channel.

The recipient can use the key and the file like that:

`./EncryptFileWithAESGCM -decrypt -file [encrypted-file] -key [hex-key]`
