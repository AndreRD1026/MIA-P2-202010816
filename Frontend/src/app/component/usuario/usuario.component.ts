import { Component } from '@angular/core';
import { Router } from '@angular/router';
import { ActivatedRoute } from '@angular/router';
import { ProyectoService } from 'src/app/services/proyecto.service';

@Component({
  selector: 'app-usuario',
  templateUrl: './usuario.component.html',
  styleUrls: ['./usuario.component.css']
})
export class UsuarioComponent {

  idparticion: string;
  user: string;


  constructor(private service: ProyectoService, private router : Router, private route: ActivatedRoute) { 
    this.idparticion = ""
    this.user = ""
  }

  ngOnInit(): void {
    this.route.queryParams.subscribe(params => {
      this.idparticion = params['idparticion'];
      this.user = params['user'];
    });
  }


  cerrarSesion(){
    this.service.postLogout().subscribe(async (res: any) => {
      this.router.navigate(['/login'])
    });
  }

  prueba(){
    console.log("Que sale? ", this.idparticion );
    
  }

}