import { Directive, ElementRef, Renderer } from '@angular/core';
		
@Directive({
  selector: '[autofocus]' // using [ ] means selecting attributes		
})
export class Autofocus {
  
  constructor(private renderer: Renderer, private el: ElementRef) { }

  ngOnInit() {
    this.renderer.invokeElementMethod(this.el.nativeElement, 'focus');
  }
}
