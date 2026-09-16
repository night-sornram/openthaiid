#import "tray_darwin.h"
#import <Cocoa/Cocoa.h>

extern void openthaiidTrayAction(int item);
extern int openthaiidTrayAutostartOn(void);

@interface OpenThaiIDTray : NSObject
@property(strong) NSStatusItem *item;
- (void)setupWithPNG:(NSData *)png;
- (void)ready:(id)sender;
- (void)technician:(id)sender;
- (void)status:(id)sender;
- (void)autostart:(id)sender;
- (void)update:(id)sender;
- (void)releases:(id)sender;
- (void)quit:(id)sender;
@end

@implementation OpenThaiIDTray
- (void)setupWithPNG:(NSData *)png {
  NSStatusBar *bar = [NSStatusBar systemStatusBar];
  self.item = [bar statusItemWithLength:NSSquareStatusItemLength];
  self.item.button.toolTip = @"ตัวช่วย OpenThaiID";
  if (png.length > 0) {
    NSImage *image = [[NSImage alloc] initWithData:png];
    if (image) {
      image.size = NSMakeSize(18, 18);
      image.template = YES;
      self.item.button.image = image;
    }
  }
  if (self.item.button.image == nil) {
    self.item.button.title = @"ID";
  }
  NSMenu *menu = [[NSMenu alloc] init];
  [[menu addItemWithTitle:@"เปิดหน้า พร้อม" action:@selector(ready:) keyEquivalent:@""] setTarget:self];
  [[menu addItemWithTitle:@"Technician tools" action:@selector(technician:) keyEquivalent:@""] setTarget:self];
  [[menu addItemWithTitle:@"แสดงสถานะ" action:@selector(status:) keyEquivalent:@""] setTarget:self];
  [menu addItem:[NSMenuItem separatorItem]];
  NSMenuItem *autoItem = [[NSMenuItem alloc] initWithTitle:@"เริ่มเมื่อเข้าสู่ระบบ"
                                                    action:@selector(autostart:)
                                             keyEquivalent:@""];
  autoItem.target = self;
	[menu addItem:autoItem];
  [menu addItem:[NSMenuItem separatorItem]];
  [[menu addItemWithTitle:@"Check for updates" action:@selector(update:) keyEquivalent:@""] setTarget:self];
  [[menu addItemWithTitle:@"GitHub Releases" action:@selector(releases:) keyEquivalent:@""] setTarget:self];
  [menu addItem:[NSMenuItem separatorItem]];
  [[menu addItemWithTitle:@"ออก" action:@selector(quit:) keyEquivalent:@"q"] setTarget:self];
  self.item.menu = menu;
}
- (void)ready:(id)sender {
  (void)sender;
  openthaiidTrayAction(1);
}
- (void)technician:(id)sender {
  (void)sender;
  openthaiidTrayAction(2);
}
- (void)status:(id)sender {
  (void)sender;
  openthaiidTrayAction(3);
}
- (void)autostart:(id)sender {
  (void)sender;
  openthaiidTrayAction(4);
}
- (void)update:(id)sender {
  (void)sender;
  openthaiidTrayAction(6);
}
- (void)releases:(id)sender {
  (void)sender;
  openthaiidTrayAction(7);
}
- (void)quit:(id)sender {
  (void)sender;
  openthaiidTrayAction(5);
}
- (BOOL)validateMenuItem:(NSMenuItem *)menuItem {
  if (menuItem.action == @selector(autostart:)) {
    menuItem.state = openthaiidTrayAutostartOn() ? NSControlStateValueOn : NSControlStateValueOff;
  }
  return YES;
}
@end

static OpenThaiIDTray *gTray;

void openthaiidDarwinTrayRun(const unsigned char *png, int pngLen) {
  @autoreleasepool {
    [NSApplication sharedApplication];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
    NSData *data = nil;
    if (png != NULL && pngLen > 0) {
      data = [NSData dataWithBytes:png length:(NSUInteger)pngLen];
    }
    gTray = [[OpenThaiIDTray alloc] init];
    [gTray setupWithPNG:data];
    [NSApp run];
  }
}

void openthaiidDarwinTrayQuit(void) {
  dispatch_async(dispatch_get_main_queue(), ^{
    [NSApp stop:nil];
    NSEvent *event = [NSEvent otherEventWithType:NSEventTypeApplicationDefined
                                        location:NSMakePoint(0, 0)
                                   modifierFlags:0
                                       timestamp:0
                                    windowNumber:0
                                         context:nil
                                         subtype:0
                                           data1:0
                                           data2:0];
    [NSApp postEvent:event atStart:YES];
  });
}

void openthaiidDarwinAlert(const char *title, const char *body) {
  NSString *t = title ? [NSString stringWithUTF8String:title] : @"OpenThaiID";
  NSString *b = body ? [NSString stringWithUTF8String:body] : @"";
  NSAlert *alert = [[NSAlert alloc] init];
  alert.messageText = t;
  alert.informativeText = b;
  alert.alertStyle = NSAlertStyleInformational;
  [alert runModal];
}
