import { provideHttpClient } from '@angular/common/http';
import { bootstrapApplication } from '@angular/platform-browser';

import { Tracker } from './tracker';

void bootstrapApplication(Tracker, { providers: [provideHttpClient()] }).catch((error: Error) => {
  console.error(error);
});
