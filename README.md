### Generate asset
text2wave -eval '(voice_msu_ru_nsh_clunits)' < textfile > wavfile.wav
echo "включи свет на кухне" | text2wave -eval '(voice_msu_ru_nsh_clunits)' -o kitchen.wav

### Record wav voice to stdout
rec -c 1 -t wav /dev/stdout rate 16k silence 1 0.1 3% 1 3.0 3% > /tmp/rec.wav


### WAV topics
- https://github.com/xlab/pocketsphinx-go/issues/3