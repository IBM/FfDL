import { Injectable } from '@angular/core';
import { Title } from '@angular/platform-browser';


@Injectable()
export class AppService {

  baseTitle = 'IBM Deep Learning Platform';
  titleSeparator = ' | ';

  constructor(private titleService: Title) {
  }

  inferTitleFromUrl(url: string) {
    const relativeUrl = url.replace(/^\/|\/$/g, '');
    let newTitle = '';
    if (relativeUrl) {
      newTitle += relativeUrl.split('/').map(word => word.length ? word[0].toUpperCase() + word.substring(1)
        : word).join(' ');
    }
    this.setTitle(newTitle);
  }

  setTitle(title: string) {
    let newTitle = '';
    if (title) {
      newTitle += `${title}${this.titleSeparator}`;
    }
    newTitle += this.baseTitle;
    this.titleService.setTitle(newTitle);
  }

  getTitle() {
    return this.titleService.getTitle();
  }

}
