import { Component } from '@angular/core';
import { Router } from '@angular/router';
import { ProyectoService } from 'src/app/services/proyecto.service';

@Component({
  selector: 'app-usuario',
  templateUrl: './usuario.component.html',
  styleUrls: ['./usuario.component.css']
})
export class UsuarioComponent {

  constructor(private service: ProyectoService, private router : Router) { }

  ngOnInit(): void {
  }


  cerrarSesion(){
    this.service.postLogout().subscribe(async (res: any) => {
      this.router.navigate(['/login'])
    });
  }

}