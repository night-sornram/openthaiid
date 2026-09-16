#ifndef OPENTHAIID_TRAY_DARWIN_H
#define OPENTHAIID_TRAY_DARWIN_H

void openthaiidDarwinTrayRun(const unsigned char *png, int pngLen);
void openthaiidDarwinTrayQuit(void);
void openthaiidDarwinAlert(const char *title, const char *body);

#endif
