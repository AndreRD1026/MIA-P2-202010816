import { Component } from '@angular/core';
import { Router } from '@angular/router';
import { ActivatedRoute } from '@angular/router';
import Swal from 'sweetalert2';
import { ProyectoService } from 'src/app/services/proyecto.service';

@Component({
  selector: 'app-usuario',
  templateUrl: './usuario.component.html',
  styleUrls: ['./usuario.component.css']
})
export class UsuarioComponent {

  idparticion: string;
  user: string;
  rutaa : string;


  constructor(private service: ProyectoService, private router : Router, private route: ActivatedRoute) { 
    this.idparticion = ""
    this.user = ""
    this.rutaa = ""
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
    console.log("Hola");
    
    console.log("Que sale? ", this.idparticion );
    
  }

  obtenerRuta(){
    console.log("SI? " , this.rutaa);
  }

  generarDisk(){
    if (!this.rutaa) {
      Swal.fire({
        title: 'Error',
        text: 'Debes proporcionar una ruta para generar este reporte',
        icon: 'error',
      });
      return;
    }
    console.log("Prueba", this.rutaa);
    
  }

}