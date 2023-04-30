import { Component, OnInit } from '@angular/core';
import { Router } from '@angular/router';
import { ActivatedRoute } from '@angular/router';
import { DomSanitizer } from '@angular/platform-browser';
import { ProyectoService } from 'src/app/services/proyecto.service';

@Component({
  selector: 'app-reportes',
  templateUrl: './reportes.component.html',
  styleUrls: ['./reportes.component.css']
})
export class ReportesComponent implements OnInit {

  imagePath: any;
  obtenerrespuesta : string;


  constructor(private _sanitizer: DomSanitizer,private router : Router ,private route: ActivatedRoute,private service: ProyectoService){
    this.obtenerrespuesta = ""
  }

  ngOnInit(): void {
    this.route.queryParams.subscribe(params => {
      this.obtenerrespuesta = params['pasarrepuesta'];

      let img = JSON.parse(JSON.stringify(this.obtenerrespuesta))
      this.imagePath = this._sanitizer.bypassSecurityTrustResourceUrl(img);
    });
  }

}
